from flask import Flask, request, Request, render_template, make_response, jsonify
import os
import logging
import re
import json
import base64
from typing import List

app = Flask(__name__)
app.template_folder = os.path.join(os.path.dirname(__file__), "templates")

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

@app.before_request
def log_request_info():
    logger.info(f"Request: {request.method} {request.path}\n  - Query: {request.args}\n  - Headers: {request.headers}\n  - Body: {request.get_data()}")

class GetMatcherConfig:

    _ROOT_PATH_CFG: List[dict] = {}

    @staticmethod
    def load_config_from_file(file_path):
        try:
            with open(file_path, 'r') as f:
                GetMatcherConfig._ROOT_PATH_CFG = json.load(f)
        except Exception as e:
            logger.error(f"Failed to load configuration: {e}")

    def __init__(self):
        config_path = os.path.join(os.path.dirname(__file__), "expectations.json")
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
        body_conditions = entry.get('httpRequest', {}).get('body', {})
        
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
        for k, v in entry.get('httpRequest', {}).get('headers', {}).items():
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
        method = cfg.get('httpRequest', {}).get( "method", '')
        if not method:
            return True
        return req.method.lower() == method.lower()
    
    def _is_path_match(self, req: Request, cfg: dict) -> bool:
        path = cfg.get('httpRequest', {}).get("path", '')
        if not path:
            return True
        return req.path == path

    
    def match_route(self, req: Request) -> dict:
        matching_routes = []

        for i in range(len(self._ROOT_PATH_CFG)):
            route_name: str = f"route_{i}"
            cfg: dict = self._ROOT_PATH_CFG[i]
            logger.debug(f"Evaluating route: {route_name}")

            is_method_match: bool = self._is_method_match(req, cfg)
            if not is_method_match:
                logger.debug(f"Method mismatch for route {route_name}")
                continue

            is_query_match: bool = self._match_json_by_key(req.args, cfg.get('httpRequest', {}).get("queryStringParameters", {}))
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
                        "Content-Type": ["application/json"]
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
config_path = os.path.join(os.path.dirname(__file__), "expectations.json")
cfg_obj: GetMatcherConfig = GetMatcherConfig()

@app.route('/api/2.0/accounts/<my_account_id>/workspaces', methods=['GET'])
def workspace_list_eccentric(my_account_id: str):
    if my_account_id != "contrivedID":
        return jsonify({'error': f'Invalid account: {my_account_id}'}), 404
    next_page_token = request.args.get('nextPageToken')
    if next_page_token:
        ## is the guard working?
        return jsonify({'message': f'disallowed nextPageToken = {next_page_token}'}), 404 
    rv = make_response(render_template('workspaces-list-01.json', request=request))
    rv.headers['Content-Type'] = 'text/plain' # This is key to appropriate testing
    rv.status_code = 200
    return rv

# A catch-all route that accepts any path
@app.route('/<path:any_path>', methods=['GET', 'POST', 'PUT', 'DELETE', 'PATCH'])
def catch_all(any_path):
    return generic_handler(request)

def generic_handler(request: Request):
    """Route POST requests to the correct template based on mockserver rules."""
    route_cfg: dict = cfg_obj.match_route(request)
    template_name = route_cfg.get("httpResponse", {}).get("template", "")
    if not template_name:
        logger.error(f"Missing template for route: {request}")
        return jsonify({'error': f'Missing template for route: {request}'}), 500
    logger.info(f"routing to template: {template_name}")
    response = make_response(render_template(template_name, request=request))
    response.headers.update(route_cfg.get("httpResponse", {}).get("headers", {}))
    response.status_code = route_cfg.get("httpResponse", {}).get("status", 200)
    return response

if __name__ == "__main__":
    app.run(debug=True, host="0.0.0.0", port=5000)
