
import logging
from flask import Flask, render_template, request, jsonify
import json

import os

app = Flask(__name__)

_IS_DOCKER = True if os.getenv('IS_DOCKER', 'false').lower() == 'true' else False

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

@app.before_request
def log_request_info():
    logger.info(f"Request: {request.method} {request.path} - Query: {request.args} -- Body: {request.get_data(as_text=True)}")

@app.route('/storage/v1/b', methods=['GET'])
def v1_storage_buckets_list():
    if request.args.get('project') == 'stackql-demo':
        return render_template('buckets-list.json'), 200, {'Content-Type': 'application/json'}
    return '{"msg": "Project Not Found"}', 404, {'Content-Type': 'application/json'}

@app.route('/storage/v1/b', methods=['POST'])
def v1_storage_buckets_insert():
    # Validate the incoming query
    body = request.get_json()
    if not body or 'name' not in body:
        return '{"msg": "Invalid request body"}', 400, {'Content-Type': 'application/json'}
    bucket_name = body['name']
    project_name = request.args.get('project')
    if not project_name:
        return '{"msg": "Invalid request: project not supplied"}', 400, {'Content-Type': 'application/json'}
    if project_name == 'testing-project':
        return render_template('buckets-insert-generic.jinja.json', bucket_name=bucket_name), 200, {'Content-Type': 'application/json'}
    return '{"msg": "Disallowed"}', 401, {'Content-Type': 'application/json'}

@app.route('/storage/v1/b/<bucket_name>', methods=['PATCH'])
def v1_storage_buckets_update(bucket_name: str):
    body = request.get_json()
    labels = json.dumps(body.get('labels', {}))
    return render_template('buckets-update-generic.jinja.json', bucket_name=bucket_name, labels=labels), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/global/networks', methods=['GET'])
def projects_testing_project_global_networks():
    return render_template('route_27_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/networks', methods=['POST'])
def compute_networks_insert(project_name: str):
    # Validate the incoming query
    body = request.get_json()
    operation_id = '1000000000001'
    operation_name = 'operation-100000000001-10000000001-10000001-10000001'
    network_name = body['name']
    host_name = 'host.docker.internal' if _IS_DOCKER else 'localhost'
    target_link = f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/networks/{ network_name }'
    if not body or 'name' not in body:
        return '{"msg": "Invalid request body"}', 400, {'Content-Type': 'application/json'}
    if not project_name:
        return '{"msg": "Invalid request: project not supplied"}', 400, {'Content-Type': 'application/json'}
    if project_name == 'mutable-project' and network_name == 'auto-test-01':
        return render_template(
            'global-operation.jinja.json', 
            target_link=target_link, 
            operation_id=operation_id,
            operation_name=operation_name,
            project_name=project_name,
            host_name=host_name,
            kind='compute#operation',
            operation_type='insert',
            progress=0,
        ), 200, {'Content-Type': 'application/json'}
    return '{"msg": "Disallowed"}', 401, {'Content-Type': 'application/json'}

def _extrapolate_target_from_operation(operation_name: str, project_name: str, host_name: str) -> str:
    """
    Extrapolates the target link from the operation name and project name.
    """
    if project_name == 'mutable-project' and operation_name == 'operation-100000000001-10000000001-10000001-10000001':
        network_name = 'auto-test-01'
        return f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/networks/{ network_name }'
    if project_name == 'mutable-project' and operation_name == 'operation-100000000002-10000000002-10000002-10000002':
        firewall_name = 'replacable-firewall'
        return f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    if project_name == 'mutable-project' and operation_name == 'operation-100000000003-10000000003-10000003-10000003':
        firewall_name = 'updatable-firewall'
        return f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    if project_name == 'mutable-project' and operation_name == 'operation-100000000004-10000000004-10000004-10000004':
        firewall_name = 'deletable-firewall'
        return f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    raise ValueError(f"Unsupported operation name: {operation_name} for project: {project_name}")

@app.route('/compute/v1/projects/<project_name>/global/operations/<operation_name>', methods=['GET'])
def projects_testing_project_global_operation_detail(project_name: str, operation_name: str):
    try:
        operation_id = '1000000000001'
        host_name = 'host.docker.internal' if _IS_DOCKER else 'localhost'
        target_link = _extrapolate_target_from_operation(operation_name, project_name, host_name)
        return render_template(
            'global-operation.jinja.json',
            target_link=target_link, 
            operation_id=operation_id,
            operation_name=operation_name,
            project_name=project_name,
            host_name=host_name,
            kind='compute#operation',
            operation_type='insert' if operation_name == 'operation-100000000001-10000000001-10000001-10000001' else 'update' if operation_name == 'operation-100000000002-10000000002-10000002-10000002' else 'delete' if operation_name == 'operation-100000000004-10000000004-10000004-10000004' else 'patch',
            progress=100,
            end_time='2025-07-05T19:43:34.491-07:00',
        ), 200, {'Content-Type': 'application/json'}
    except:
        return '{"msg": "Disallowed"}', 401, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/networks/<network_name>', methods=['GET'])
def projects_testing_project_global_network_detail(project_name: str, network_name: str):
    return render_template(
        'networks-insert-generic-mature.jinja.json', 
        project_name=project_name, 
        network_name=network_name
    ), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-three/locations/global/keyRings/testing-three/cryptoKeys', methods=['GET'])
def v1_projects_testing_project_three_locations_global_keyRings_testing_three_cryptoKeys():
    return render_template('route_1_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-three/locations/australia-southeast2/keyRings/big-m-testing-three/cryptoKeys', methods=['GET'])
def v1_projects_testing_project_three_locations_australia_southeast2_keyRings_big_m_testing_three_cryptoKeys():
    return render_template('route_2_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-three/locations/australia-southeast1/keyRings', methods=['GET'])
def v1_projects_testing_project_three_locations_australia_southeast1_keyRings():
    return render_template('route_3_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-three/locations/global/keyRings', methods=['GET'])
def v1_projects_testing_project_three_locations_global_keyRings():
    return render_template('route_4_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-three/locations/australia-southeast2/keyRings', methods=['GET'])
def v1_projects_testing_project_three_locations_australia_southeast2_keyRings():
    return render_template('route_5_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-two/locations/global/keyRings/testing-two/cryptoKeys', methods=['GET'])
def v1_projects_testing_project_two_locations_global_keyRings_testing_two_cryptoKeys():
    return render_template('route_6_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-two/locations/australia-southeast2/keyRings/big-m-testing-two/cryptoKeys', methods=['GET'])
def v1_projects_testing_project_two_locations_australia_southeast2_keyRings_big_m_testing_two_cryptoKeys():
    return render_template('route_7_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-two/locations/australia-southeast1/keyRings', methods=['GET'])
def v1_projects_testing_project_two_locations_australia_southeast1_keyRings():
    return render_template('route_8_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-two/locations/global/keyRings', methods=['GET'])
def v1_projects_testing_project_two_locations_global_keyRings():
    return render_template('route_9_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project-two/locations/australia-southeast2/keyRings', methods=['GET'])
def v1_projects_testing_project_two_locations_australia_southeast2_keyRings():
    return render_template('route_10_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project/locations/global/keyRings/testing/cryptoKeys', methods=['GET'])
def v1_projects_testing_project_locations_global_keyRings_testing_cryptoKeys():
    return render_template('route_11_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project/locations/australia-southeast1/keyRings', methods=['GET'])
def v1_projects_testing_project_locations_australia_southeast1_keyRings():
    return render_template('route_12_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project/locations/global/keyRings', methods=['GET'])
def v1_projects_testing_project_locations_global_keyRings():
    return render_template('route_13_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/instances/000000001/getIamPolicy', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_instances_000000001_getIamPolicy():
    return render_template('route_14_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/machineTypes', methods=['GET'])
def machine_types():
    import math

    # The full list of machine type names
    machine_type_names = [
        f"p{page}-c2-standard-{vcpus}"
        for page in range(1, 21)  # Pages 1 through 20
        for vcpus in [8, 60, 4, 30, 16]
    ]

    # Pagination parameters
    items_per_page = 10  # Adjust as needed
    current_page = int(request.args.get('pageToken', 1))  # Default to page 1
    total_pages = math.ceil(len(machine_type_names) / items_per_page)

    # Determine start and end indices for the current page
    start_idx = (current_page - 1) * items_per_page
    end_idx = start_idx + items_per_page

    # Slice the list for the current page
    page_items = machine_type_names[start_idx:end_idx]

    # Prepare the response
    response = {
        "kind": "compute#machineTypeList",
        "id": "projects/testing-project/zones/australia-southeast1-a/machineTypes",
        "items": [
            {
                "kind": "compute#machineType",
                "id": f"8010{page_items.index(name)}",
                "name": name,
                "description": f"{name} description",
                "guestCpus": int(name.split('-')[-1]),
                "memoryMb": int(name.split('-')[-1]) * 4096,  # Example calculation
                "zone": "australia-southeast1-a",
                "selfLink": f"https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a/machineTypes/{name}",
            }
            for name in page_items
        ],
        "selfLink": "https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a/machineTypes"
    }

    # Add nextPageToken if applicable
    if current_page < total_pages:
        response["nextPageToken"] = str(current_page + 1)

    return jsonify(response)



## TODO: geet rid once all else stable
@app.route('/token', methods=['POST'])
def token():
    return render_template('route_16_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/testing-project/aggregated/usableSubnetworks', methods=['GET'])
def v1_projects_testing_project_aggregated_usableSubnetworks():
    return render_template('route_17_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/another-project/aggregated/usableSubnetworks', methods=['GET'])
def v1_projects_another_project_aggregated_usableSubnetworks():
    return render_template('route_18_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/yet-another-project/aggregated/usableSubnetworks', methods=['GET'])
def v1_projects_yet_another_project_aggregated_usableSubnetworks():
    return render_template('route_19_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v1/projects/empty-project/aggregated/usableSubnetworks', methods=['GET'])
def v1_projects_empty_project_aggregated_usableSubnetworks():
    return render_template('route_20_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/acceleratorTypes', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_acceleratorTypes():
    return render_template('route_21_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/defective-response-content-project/zones/australia-southeast1-a/acceleratorTypes', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_acceleratorTypes_defective_response_type():
    return render_template('defective-content-type-accelerator-type-list.json'), 200, {'Content-Type': 'text/plain'}

@app.route('/compute/v1/projects/another-project/zones/australia-southeast1-a/acceleratorTypes', methods=['GET'])
def projects_another_project_zones_australia_southeast1_a_acceleratorTypes():
    return render_template('route_22_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v3/projects/testproject:getIamPolicy', methods=['POST'])
def v3_projects_testproject_getIamPolicy():
    return render_template('route_23_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/v3/organizations/123456789012:getIamPolicy', methods=['POST'])
def v3_organizations_123456789012_getIamPolicy():
    return render_template('route_24_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/disks', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_disks():
    return render_template('route_25_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-b/disks', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_b_disks():
    return render_template('route_26_template.json'), 200, {'Content-Type': 'application/json'}


@app.route('/compute/v1/projects/testing-project/regions/australia-southeast1/subnetworks', methods=['GET'])
def projects_testing_project_regions_australia_southeast1_subnetworks():
    return render_template('route_28_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/instances', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_instances():
    return render_template('route_29_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/aggregated/instances', methods=['GET'])
def projects_testing_project_aggregated_instances():
    return render_template('route_30_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/stackql-interesting/aggregated/instances', methods=['GET'])
def projects_stackql_interesting_aggregated_instances():
    return render_template('instances-agg-list-interesting.jinja.json'), 200, {'Content-Type': 'application/json'}

assets_counter = {'count': 0}
@app.route('/v1/projects/testing-project/assets', methods=['GET'])
def v1_projects_testing_project_assets():
    next_page_token = request.args.get('pageToken', )
    if next_page_token == 'GETAROUNDPAGETWO':
        return render_template('route_31_template.json'), 200, {'Content-Type': 'application/json'}
    # Increment the call counter
    return render_template('route_32_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/firewalls/<firewall_name>', methods=['PUT'])
def projects_testing_project_global_firewalls_replace(project_name: str, firewall_name: str):
    _permitted_combinations = (('testing-project', 'allow-spark-ui'), ('mutable-project', 'replacable-firewall'))
    if (project_name, firewall_name) not in _permitted_combinations:
        return '{"msg": "Disallowed"}', 500, {'Content-Type': 'application/json'}
    body = request.get_json()
    operation_id = '1000000000002'
    operation_name = 'operation-100000000002-10000000002-10000002-10000002'
    host_name = 'host.docker.internal' if _IS_DOCKER else 'localhost'
    target_link = f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    if not body:
        return '{"msg": "Invalid request body"}', 400, {'Content-Type': 'application/json'}
    if not project_name:
        return '{"msg": "Invalid request: project not supplied"}', 400, {'Content-Type': 'application/json'}
    return render_template(
        'global-operation.jinja.json', 
        target_link=target_link, 
        operation_id=operation_id,
        operation_name=operation_name,
        project_name=project_name,
        host_name=host_name,
        kind='compute#operation',
        operation_type='update',
        progress=0,
    ), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/firewalls/<firewall_name>', methods=['PATCH'])
def projects_testing_project_global_firewalls_update(project_name: str, firewall_name: str):
    _permitted_combinations = (('testing-project', 'some-other-firewall'), ('mutable-project', 'updatable-firewall'))
    if (project_name, firewall_name) not in _permitted_combinations:
        return '{"msg": "Disallowed"}', 500, {'Content-Type': 'application/json'}
    body = request.get_json()
    operation_id = '1000000000003'
    operation_name = 'operation-100000000003-10000000003-10000003-10000003'
    host_name = 'host.docker.internal' if _IS_DOCKER else 'localhost'
    target_link = f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    if not body:
        return '{"msg": "Invalid request body"}', 400, {'Content-Type': 'application/json'}
    if not project_name:
        return '{"msg": "Invalid request: project not supplied"}', 400, {'Content-Type': 'application/json'}
    return render_template(
        'global-operation.jinja.json', 
        target_link=target_link, 
        operation_id=operation_id,
        operation_name=operation_name,
        project_name=project_name,
        host_name=host_name,
        kind='compute#operation',
        operation_type='patch',
        progress=0,
    ), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/firewalls/<firewall_name>', methods=['DELETE'])
def projects_testing_project_global_firewalls_delete(project_name: str, firewall_name: str):
    _permitted_combinations = (('mutable-project', 'deletable-firewall'),)
    if (project_name, firewall_name) not in _permitted_combinations:
        return '{"msg": "Disallowed"}', 500, {'Content-Type': 'application/json'}
    operation_id = '1000000000004'
    operation_name = 'operation-100000000004-10000000004-10000004-10000004'
    host_name = 'host.docker.internal' if _IS_DOCKER else 'localhost'
    target_link = f'https://{host_name}:1080/compute/v1/projects/{ project_name }/global/firewalls/{ firewall_name }'
    if not project_name:
        return '{"msg": "Invalid request: project not supplied"}', 400, {'Content-Type': 'application/json'}
    return render_template(
        'global-operation.jinja.json', 
        target_link=target_link, 
        operation_id=operation_id,
        operation_name=operation_name,
        project_name=project_name,
        host_name=host_name,
        kind='compute#operation',
        operation_type='delete',
        progress=0,
    ), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/<project_name>/global/firewalls/<firewall_name>', methods=['GET'])
def projects_testing_project_global_firewalls_some_other_firewall(project_name: str, firewall_name: str):
    _permitted_combinations = (('testing-project', 'some-other-firewall'), ('mutable-project', 'updatable-firewall'), ('mutable-project', 'replacable-firewall'))
    if (project_name, firewall_name) not in _permitted_combinations:
        return '{"msg": "Disallowed"}', 500, {'Content-Type': 'application/json'}
    jinja_context = {
        'project_name': project_name,
        'firewall_name': firewall_name,
        'host_name': 'host.docker.internal' if _IS_DOCKER else 'localhost',
    }
    return render_template('firewall-detail.jinja.json', **jinja_context), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/global/firewalls', methods=['GET'])
def projects_testing_project_global_firewalls():
    return render_template('route_35_template.json'), 200, {'Content-Type': 'application/json'}


# Initialize a counter to track the number of calls
call_counter = {"firewalls": 0}

@app.route('/compute/v1/projects/changing-project/global/firewalls', methods=['GET'])
def firewalls():
    # Increment the call counter
    call_counter["firewalls"] += 1

    # Determine the response based on the call count
    if call_counter["firewalls"] <= 2:
        return render_template('route_36_template.json'), 200, {'Content-Type': 'application/json'}
    else:
        return render_template('route_37_template.json'), 200, {'Content-Type': 'application/json'}

if __name__ == '__main__':
    app.run(debug=True)
