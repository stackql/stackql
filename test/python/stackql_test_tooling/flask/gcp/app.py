
import logging
from flask import Flask, render_template, request, jsonify

app = Flask(__name__)

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

@app.before_request
def log_request_info():
    logger.info(f"Request: {request.method} {request.path} - Query: {request.args}")

@app.route('/storage/v1/b', methods=['GET'])
def v1_storage_buckets_list():
    if request.args.get('project') == 'stackql-demo':
        return render_template('buckets-list.json'), 200, {'Content-Type': 'application/json'}
    return '{"msg": "Project Not Found"}', 404, {'Content-Type': 'application/json'}

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

@app.route('/compute/v1/projects/testing-project/global/networks', methods=['GET'])
def projects_testing_project_global_networks():
    return render_template('route_27_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/regions/australia-southeast1/subnetworks', methods=['GET'])
def projects_testing_project_regions_australia_southeast1_subnetworks():
    return render_template('route_28_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/zones/australia-southeast1-a/instances', methods=['GET'])
def projects_testing_project_zones_australia_southeast1_a_instances():
    return render_template('route_29_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/aggregated/instances', methods=['GET'])
def projects_testing_project_aggregated_instances():
    return render_template('route_30_template.json'), 200, {'Content-Type': 'application/json'}

assets_counter = {'count': 0}
@app.route('/v1/projects/testing-project/assets', methods=['GET'])
def v1_projects_testing_project_assets():
    next_page_token = request.args.get('pageToken', )
    if next_page_token == 'GETAROUNDPAGETWO':
        return render_template('route_31_template.json'), 200, {'Content-Type': 'application/json'}
    # Increment the call counter
    return render_template('route_32_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/global/firewalls/allow-spark-ui', methods=['PUT'])
def projects_testing_project_global_firewalls_allow_spark_ui():
    return render_template('route_33_template.json'), 200, {'Content-Type': 'application/json'}

@app.route('/compute/v1/projects/testing-project/global/firewalls/some-other-firewall', methods=['PATCH'])
def projects_testing_project_global_firewalls_some_other_firewall():
    return render_template('route_34_template.json'), 200, {'Content-Type': 'application/json'}

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
