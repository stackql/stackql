from flask import Flask, request, Request, render_template, make_response, jsonify
import os
import logging
import re
import json
import base64
import datetime

app = Flask(__name__)
app.template_folder = os.path.join(os.path.dirname(__file__), "templates")

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

@app.before_request
def log_request_info():
    logger.info(f"Request: {request.method} {request.path}\n  - Query: {request.args}\n  - Headers: {request.headers}\n  - Body: {request.get_data()}")

class GetMatcherConfig:

    _ROOT_PATH_CFG: dict = {}

    @staticmethod
    def load_config_from_file(file_path):
        try:
            with open(file_path, 'r') as f:
                GetMatcherConfig._ROOT_PATH_CFG = json.load(f)

                # Decode base64 responses in templates
                for route_name, cfg in GetMatcherConfig._ROOT_PATH_CFG.items():
                    if "base64_template" in cfg:
                        try:
                            decoded_content = base64.b64decode(cfg["base64_template"]).decode("utf-8")
                            template_path = os.path.join(app.template_folder, cfg["template"]) 
                            with open(template_path, "w") as tpl_file:
                                tpl_file.write(decoded_content)
                            logger.info(f"Decoded base64 template for route: {route_name}")
                        except Exception as e:
                            logger.error(f"Failed to decode base64 template for route: {route_name}: {e}")

                logger.info("Configuration loaded and templates processed successfully.")
        except Exception as e:
            logger.error(f"Failed to load configuration: {e}")

    def __init__(self):
        config_path = os.path.join(os.path.dirname(__file__), "root_path_cfg.json")
        self.load_config_from_file(config_path)

    @staticmethod
    def get_config(path_name):
        return GetMatcherConfig._ROOT_PATH_CFG.get(path_name, None)
    
    def _match_json_strict(self, lhs: dict, rhs: dict) -> bool:
        matches = json.dumps(
            lhs, sort_keys=True) == json.dumps(
                rhs, sort_keys=True)
        return matches
    
    def _match_json_by_key(self, lhs: dict, rhs: dict) -> bool:
        for key, value in rhs.items():
            if key not in lhs:
                return False
            if isinstance(value, dict):
                if not self._match_json_by_key(lhs[key], value):
                    return False
            elif isinstance(value, list):
                for item in value:
                    if not self._match_string(lhs[key], item):
                        return False
            elif isinstance(value, str):
                if not self._match_string(lhs[key], value):
                    return False
            else:
                if lhs[key] != value:
                    return False
        return True
    
    def _match_json_request_body(self, lhs: dict, rhs: dict, match_type: str) -> bool:
        if match_type.lower() == 'strict':
            return self._match_json_strict(lhs, rhs)
        elif match_type.lower() == 'only_matching_fields':
            return self._match_json_by_key(lhs, rhs)
        return False
    
    def _match_request_body(self, req: Request, entry: dict) -> bool:
        body_conditions = entry.get('body_conditions', {})
        
        if not body_conditions:
            return True
    
        logger.warning('evaluating body conditions')
        json_body = body_conditions.get('json', {})
        if json_body:
            request_body = request.get_json(silent=True, force=True)
            logger.debug(f'comparing expected body = {json_body}, with request body = {request_body}')
            if json_body:
                return self._match_json_request_body(request_body, json_body, body_conditions.get('matchType', 'strict'))
        form_body = body_conditions.get('parameters', {})
        if form_body:
            request_body = request.form
            logger.debug(f'comparing expected body = {form_body}, with request body = {request_body}')
            return self._match_json_by_key(request_body, form_body)
        string_body = body_conditions.get('type', '').lower() == 'string'
        if string_body:
            request_body = request.get_data(as_text=True)
            logger.warning(f'comparing expected body = {body_conditions.get("value")}, with request body = {request_body}')
            return self._match_string(request_body, body_conditions.get('value'))
        return False
    
    def _match_string(self, lhs: str, rhs: str) -> bool:
        if lhs == rhs:
            return True
        if re.match(rhs, lhs):
            return True
        return False

    def _match_request_headers(self, req: Request, entry: dict) -> bool:
        for k, v in entry.get('headers', {}).items():
            if type(v) == str:
                if not self._match_string(req.headers.get(k), v):
                    return False
            elif type(v) == list:
                ## Could make thi smore complex if needed
                match_found = False
                for item in v:
                    if self._match_string(req.headers.get(k), item):
                        match_found = True
                        break
                if not match_found:
                    return False
        return True
    
    def _is_method_match(self, req: Request, cfg: dict) -> bool:
        method = cfg.get("method", '')
        if not method:
            return True
        return req.method.lower() == method.lower()
    
    def _is_path_match(self, req: Request, cfg: dict) -> bool:
        path = cfg.get("path", '')
        if not path:
            return True
        return req.path == path

    
    def match_route(self, req: Request) -> dict:
        matching_routes = []

        for route_name, cfg in self._ROOT_PATH_CFG.items():
            logger.debug(f"Evaluating route: {route_name}")

            is_method_match: bool = self._is_method_match(req, cfg)
            if not is_method_match:
                logger.debug(f"Method mismatch for route {route_name}")
                continue

            is_query_match: bool = self._match_json_by_key(req.args, cfg.get("queryStringParameters", {}))
            if not is_query_match:
                logger.debug(f"Query mismatch for route {route_name}")
                continue

            is_path_match: bool = self._is_path_match(req, cfg)
            if not is_path_match:
                logger.debug(f"Path mismatch for route {route_name}")
                continue
            
            is_header_match: bool = self._match_request_headers(req, cfg)
            if not is_header_match:
                logger.debug(f"Header mismatch for route {route_name}")
                continue

            is_body_match: bool = self._match_request_body(req, cfg)
            if not is_body_match:
                logger.warning(f"Body mismatch for route {route_name}")
                continue

            matching_routes.append((route_name, cfg))

        # Prioritize routes with body conditions
        matching_routes.sort(key=lambda x: bool(x[1].get("body_conditions")), reverse=True)

        if not matching_routes:
            data = req.get_data()
            logger.warning(f"No matching route found for request: {req} with {data}")
            if data == b'{"DesiredState":"{\\"BucketName\\":\\"my-bucket\\",\\"ObjectLockEnabled\\":true,\\"Tags\\":[{\\"Key\\":\\"somekey\\",\\"Value\\":\\"v4\\"}]}","TypeName":"AWS::S3::Bucket"}':
                return {
                    "template": "template_71.json",
                    "status": 200,
                    "response_headers": {
                        "Content-Type": ["application/x-amz-json-1.0"]
                    }
                }
            else:
                return {
                    "template": "empty-response.json",
                    "status": 404
                }

        if matching_routes:
            selected_route, cfg = matching_routes[0]
        return cfg
    


# Load the configuration at startup
config_path = os.path.join(os.path.dirname(__file__), "root_path_cfg.json")
cfg_obj: GetMatcherConfig = GetMatcherConfig()

# Routes generated from mockserver configuration
@app.route('/', methods=['POST', "GET"])
def handle_root_requests():
    return generic_handler(request)

@app.route('/2013-04-01/hostedzone/<rrset_id>/rrset/', methods=['POST', 'GET'])
def handle_rrset_requests(rrset_id: str):
    return generic_handler(request)

@app.route('/2013-04-01/hostedzone/<rrset_id>/rrset', methods=['GET'])
def handle_rrset_requests_unterminated(rrset_id: str):
    return generic_handler(request)

def generic_handler(request: Request):
    """Route POST requests to the correct template based on mockserver rules."""
    route_cfg: dict = cfg_obj.match_route(request)
    if "template" not in route_cfg:
        logger.error(f"Missing template for route: {request}")
        return jsonify({'error': f'Missing template for route: {request}'}), 500
    logger.info(f"routing to template: {route_cfg['template']}")
    twelve_days_ago = (datetime.datetime.now() - datetime.timedelta(days=12)).strftime("%Y-%m-%d")
    response = make_response(render_template(route_cfg["template"], request=request, twelve_days_ago=twelve_days_ago))
    response.headers.update(route_cfg.get("response_headers", {}))
    response.status_code = route_cfg.get("status", 200)
    return response

if __name__ == "__main__":
    app.run(debug=True, host="0.0.0.0", port=5000)
