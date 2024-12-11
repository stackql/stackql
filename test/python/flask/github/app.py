
import logging
from flask import Flask, request, jsonify, render_template, url_for
from datetime import datetime
import re
import json
import os

app = Flask(__name__)

templates_dir = os.path.join(os.path.dirname(__file__), 'templates')

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

# Utility function to get ISO 8601 datetime
def get_iso_8601():
    return datetime.utcnow().isoformat() + 'Z'

# Global variables for cycling users
current_page = 1

def graphql_users():
    """Retrieve users divided across pages for /graphql"""
    return [
        {"nameId": "sherlockholmes1@gmail.com", "username": "sherlockholmes1@gmail.com", "login": "some-jimbo-1"},
        {"nameId": "sherlockholmes2@gmail.com", "username": "sherlockholmes2@gmail.com", "login": "some-jimbo-2"},
        {"nameId": "sherlockholmes3@gmail.com", "username": "sherlockholmes3@gmail.com", "login": "some-jimbo-3"},
        {"nameId": "sherlockholmes4@gmail.com", "username": "sherlockholmes4@gmail.com", "login": "some-jimbo-4"},
        {"nameId": "sherlockholmes5@gmail.com", "username": "sherlockholmes5@gmail.com", "login": "some-jimbo-5"},
        {"nameId": "sherlockholmes6@gmail.com", "username": "sherlockholmes6@gmail.com", "login": "some-jimbo-6"},
        {"nameId": "sherlockholmes7@gmail.com", "username": "sherlockholmes7@gmail.com", "login": "some-jimbo-7"},
        {"nameId": "sherlockholmes8@gmail.com", "username": "sherlockholmes8@gmail.com", "login": "some-jimbo-8"},
        {"nameId": "sherlockholmes9@gmail.com", "username": "sherlockholmes9@gmail.com", "login": "some-jimbo-9"}
    ]

@app.before_request
def log_request_info():
    logger.info(f"Request: {request.method} {request.path} \nQuery Params: {request.args}")

@app.route('/graphql', methods=['POST'])
def graphql():
    global current_page
    users_per_request = 3
    total_users = len(graphql_users())

    # Validate the incoming query
    body = request.get_json()
    query_match = re.search(r'.*samlIdentityProvider.*', body.get('query', ''))
    if not query_match:
        logger.warning("Invalid query format received for /graphql")
        return jsonify({"error": "Invalid query format"}), 400

    # Calculate users to return for the current request
    all_users = graphql_users()
    guids = [
        "4d8ca614-05b3-11ed-8b3b-954c779e608e",
        "4d8ca614-05b3-11ed-8b3b-954c779e608e",
        "2b03a0f4-05bb-11ed-9795-a2b270ddb7b3"
    ]
    start_index = (current_page - 1) * users_per_request
    end_index = start_index + users_per_request
    edges = [
        {
            "node": {
                "guid": f"{guids[i%3]}",
                "samlIdentity": {
                    "nameId": user["nameId"],
                    "username": user["username"]
                },
                "scimIdentity": {
                    "username": None
                },
                "user": {
                    "login": user["login"]
                }
            },
            "cursor": f"{i+1}"
        }
        for i, user in enumerate(all_users[start_index:end_index], start=start_index)
    ]

    # Determine if there are more pages
    has_next_page = end_index <= total_users
    edges = []  if not has_next_page else edges
    response_data = {
        "data": {
            "viewer": {
                "organization": {
                    "samlIdentityProvider": {
                        "externalIdentities": {
                            "edges": edges,
                            "pageInfo": {
                                "hasNextPage": has_next_page,
                                "endCursor": edges[-1]["cursor"] if edges else None
                            }
                        }
                    }
                }
            }
        }
    }

    # Log the response and update the page for the next request
    logger.info(f"Sending /graphql response for page {current_page} with users {start_index+1} to {end_index}")
    if has_next_page:
        current_page += 1
    else:
        current_page = 1  # Reset to the first page after cycling through all users

    return jsonify(response_data), 200, {'Datetime': get_iso_8601()}


@app.route('/orgs/dummyorg', methods=['PATCH'])
def update_org():
    data = request.get_json()
    if data.get('description') == "Some silly description.":
        response = json.load(open(os.path.join(templates_dir, 'patch-dummyorg.json'), 'r'))
        logger.info("Successful /orgs/dummyorg PATCH response sent")
        return jsonify(response), 200
    logger.warning("Invalid PATCH request for /orgs/dummyorg")
    return jsonify({"error": "Invalid request"}), 400


@app.route('/users/<string:userId>', methods=['GET'])
def get_user(userId):
    """Retrieve user details dynamically from template."""
    context = {
        "login": userId,
        "id": 1,
        "node_id": "MDQ6VXNlcjE=",
        "avatar_url": "https://github.com/images/error/octocat_happy.gif",
        "gravatar_id": "",
        "url": f"https://api.github.com/users/{userId}",
        "html_url": f"https://github.com/{userId}",
        "followers_url": f"https://api.github.com/users/{userId}/followers",
        "following_url": f"https://api.github.com/users/{userId}/following{{/other_user}}",
        "gists_url": f"https://api.github.com/users/{userId}/gists{{/gist_id}}",
        "starred_url": f"https://api.github.com/users/{userId}/starred{{/owner}}{{/repo}}",
        "subscriptions_url": f"https://api.github.com/users/{userId}/subscriptions",
        "organizations_url": f"https://api.github.com/users/{userId}/orgs",
        "repos_url": f"https://api.github.com/users/{userId}/repos",
        "events_url": f"https://api.github.com/users/{userId}/events{{/privacy}}",
        "received_events_url": f"https://api.github.com/users/{userId}/received_events",
        "type": "User",
        "site_admin": False,
        "name": f"monalisa {userId}",
        "company": "GitHub",
        "blog": "https://github.com/blog",
        "location": "San Francisco",
        "email": f"{userId}@github.com",
        "hireable": False,
        "bio": "There once was...",
        "twitter_username": "monatheoctocat",
        "public_repos": 2,
        "public_gists": 1,
        "followers": 20,
        "following": 0,
        "created_at": "2008-01-14T04:33:35Z",
        "updated_at": "2008-01-14T04:33:35Z"
    }
    return render_template('user_template.json', **context), 200, {"Content-Type": "application/json"}

repos_collab_call_count = 0
repas_collab_specialcase_call_count = 0
@app.route('/repos/<string:org>/<string:repositoryId>/collaborators', methods=['GET'])
def get_aware_collaborators(org, repositoryId):
    global repos_collab_call_count
    global repas_collab_specialcase_call_count
    if org == 'specialcaseorg':
        repas_collab_specialcase_call_count += 1
        if repas_collab_specialcase_call_count == 1:
            response = json.load(open(os.path.join(templates_dir, 'first-specialcaseorg-repository-collaborators.json'), 'r'))
            logger.info(f"Collaborators sent for org: {org}, repository: {repositoryId}")
            return jsonify(response), 200
        response = json.load(open(os.path.join(templates_dir, 'specialcaseorg-repository-collaborators.json'), 'r'))
        logger.info(f"Collaborators sent for org: {org}, repository: {repositoryId}")
        return jsonify(response), 200
    else:
        repos_collab_call_count += 1
        if repos_collab_call_count == 1:
            response = json.load(open(os.path.join(templates_dir, 'first-specialcaseorg-repository-collaborators.json'), 'r'))
            logger.info(f"Collaborators sent for org: {org}, repository: {repositoryId}")
            return jsonify(response), 200
        response = json.load(open(os.path.join(templates_dir, 'specialcaseorg-repository-collaborators.json'), 'r'))
        logger.info(f"Collaborators sent for org: {org}, repository: {repositoryId}")
        return jsonify(response), 200


@app.route('/orgs/dummyorg/members', methods=['GET'])
def get_members():
    """Retrieve organization members dynamically from template."""
    page = int(request.args.get('page', 1))
    per_page = 10
    total_pages = 4
    members = []
    for i in range((page - 1) * per_page + 1, page * per_page + 1):
        members.append({
            "login": f"some-jimbo-{i}",
            "id": 1,
            "node_id": "MDQ6VXNlcjE=",
            "avatar_url": "https://github.com/images/error/octocat_happy.gif",
            "gravatar_id": "",
            "url": f"https://api.github.com/users/some-jimbo-{i}",
            "html_url": f"https://github.com/some-jimbo-{i}",
            "followers_url": f"https://api.github.com/users/some-jimbo-{i}/followers",
            "following_url": f"https://api.github.com/users/some-jimbo-{i}/following{{/other_user}}",
            "gists_url": f"https://api.github.com/users/some-jimbo-{i}/gists{{/gist_id}}",
            "starred_url": f"https://api.github.com/users/some-jimbo-{i}/starred{{/owner}}{{/repo}}",
            "subscriptions_url": f"https://api.github.com/users/some-jimbo-{i}/subscriptions",
            "organizations_url": f"https://api.github.com/users/some-jimbo-{i}/orgs",
            "repos_url": f"https://api.github.com/users/some-jimbo-{i}/repos",
            "events_url": f"https://api.github.com/users/some-jimbo-{i}/events{{/privacy}}",
            "received_events_url": f"https://api.github.com/users/some-jimbo-{i}/received_events",
            "type": "User",
            "site_admin": False
        })
    context = {
        "members": members,
        "next_page": page + 1,
        "prev_page": page - 1 if page > 1 else None
    }
    headers = {
        "Content-Type": "application/json",
        "Link": generate_pagination_links(page, total_pages, 'get_trailing_members')
    }
    return render_template('members_template.json', **context), 200, headers


@app.route('/organizations/000000001/members', methods=['GET'])
def get_trailing_members():
    """Retrieve organization members dynamically from template."""
    return get_members()


@app.route('/repos/dummyorg/dummyapp.io/tags', methods=['GET'])
def get_tags():
    """Retrieve repository tags dynamically from template."""
    page = int(request.args.get('page', 1))
    per_page = 100  # GitHub typically paginates at 100 items per page
    total_tags = 600
    total_pages = total_tags // per_page

    start_index = (page - 1) * per_page + 1
    end_index = min(page * per_page, total_tags)

    tags = [
        {
            "name": f"v{tag}",
            "zipball_url": f"https://api.github.com/repos/dummyorg/dummyapp.io/zipball/v{tag}",
            "tarball_url": f"https://api.github.com/repos/dummyorg/dummyapp.io/tarball/v{tag}",
            "commit": {
                "sha": f"sha-{tag}",
                "url": f"https://api.github.com/repos/dummyorg/dummyapp.io/commits/sha-{tag}"
            },
            "node_id": f"node-{tag}"
        } for tag in range(start_index, end_index + 1)
    ]

    context = {
        "tags": tags
    }

    headers = {
        "Content-Type": "application/json",
        "Link": generate_pagination_links(page, total_pages, 'get_tags')
    }
    return render_template('tags_template.json', **context), 200, headers

@app.route('/repositories/000000001/tags', methods=['GET'])
def get_trailing_tags():
    """Retrieve repository tags dynamically from template."""
    page = int(request.args.get('page', 1))
    per_page = 100  # GitHub typically paginates at 100 items per page
    total_tags = 600
    total_pages = total_tags // per_page

    start_index = (page - 1) * per_page + 1
    end_index = min(page * per_page, total_tags)

    tags = [
        {
            "name": f"v{tag}",
            "zipball_url": f"https://api.github.com/repos/dummyorg/dummyapp.io/zipball/v{tag}",
            "tarball_url": f"https://api.github.com/repos/dummyorg/dummyapp.io/tarball/v{tag}",
            "commit": {
                "sha": f"sha-{tag}",
                "url": f"https://api.github.com/repos/dummyorg/dummyapp.io/commits/sha-{tag}"
            },
            "node_id": f"node-{tag}"
        } for tag in range(start_index, end_index + 1)
    ]

    context = {
        "tags": tags
    }

    headers = {
        "Content-Type": "application/json",
        "Link": generate_pagination_links(page, total_pages, 'get_trailing_tags')
    }
    return render_template('tags_template.json', **context), 200, headers


def generate_pagination_links(page, total_pages, endpoint):
    """Generate GitHub-style pagination links."""
    links = []
    if page < total_pages:
        next_page = url_for(endpoint, page=page + 1, _external=True)
        links.append(f'<{next_page}>; rel="next"')
        last_page = url_for(endpoint, page=total_pages, _external=True)
        links.append(f'<{last_page}>; rel="last"')
    if page > 1:
        prev_page = url_for(endpoint, page=page - 1, _external=True)
        links.append(f'<{prev_page}>; rel="prev"')
        first_page = url_for(endpoint, page=1, _external=True)
        links.append(f'<{first_page}>; rel="first"')
    return ", ".join(links)

@app.route('/repos/dummyorg/dummyapp.io/branches', methods=['GET'])
def get_dummyorg_branches():
    """Retrieve branches for dummyorg repository with pagination."""
    page = int(request.args.get('page', 1))
    total_pages = 2
    branches = []
    if page == 1:
        branches = [
            {
                "name": "bugfix/nil-check",
                "commit": {
                    "sha": "cfd67a0711523f731de1f3d4d088a15f7e930c90",
                    "url": "https://api.github.com/repos/dummyorg/dummyapp.io/commits/cfd67a0711523f731de1f3d4d088a15f7e930c90"
                },
                "protected": False
            },
            {
                "name": "bugfix/select-schema-default",
                "commit": {
                    "sha": "8a6406b49f72bca575321bdae4b13645a64e5072",
                    "url": "https://api.github.com/repos/dummyorg/dummyapp.io/commits/8a6406b49f72bca575321bdae4b13645a64e5072"
                },
                "protected": False
            }
        ]
    elif page == 2:
        branches = [
            {
                "name": "feature/flow-control",
                "commit": {
                    "sha": "5b44c87f9f87780c6ebe9e8f455301",
                    "url": "https://api.github.com/repos/dummyorg/dummyapp.io/commits/5b44c87f9f87780c6ebe9e8f455301"
                },
                "protected": True
            },
            {
                "name": "feature/flow-control-rebased",
                "commit": {
                    "sha": "7a8d9e6f1c2d4e7b5f4c9a103b1234567890abcd",
                    "url": "https://api.github.com/repos/dummyorg/dummyapp.io/commits/7a8d9e6f1c2d4e7b5f4c9a103b1234567890abcd"
                },
                "protected": False
            }
        ]
    
    context = {
        "branches": branches,
    }

    headers = {
        "Content-Type": "application/json",
        "Link": generate_pagination_links(page, total_pages, 'get_repository_branches')
    }
    return render_template('branches_template.json', **context), 200, headers

@app.route('/repositories/000000001/branches', methods=['GET'])
def get_repository_branches():
    """Retrieve branches for repository 000000001."""
    return get_dummyorg_branches()

@app.route('/scim/v2/organizations/dummyorg/Users', methods=['GET'])
def get_scim_users():
    response = json.load(open(os.path.join(templates_dir, 'scim-dummyorg-users.json'), 'r'))
    logger.info("SCIM users data sent")
    ## need to add header for scim+json
    return json.dumps(response), 200, {"Content-Type": "application/scim+json"}


@app.route('/repos/dummyorg/dummyapp.io/pages', methods=['GET'])
def get_pages():
    response = json.load(open(os.path.join(templates_dir, 'dummyorg-dummyapp-pages.json'), 'r'))
    logger.info("Pages data sent")
    return jsonify(response), 200


@app.route('/repos/dummyorg/dummyapp.io/commits', methods=['GET'])
def get_commits():
    response = json.load(open(os.path.join(templates_dir, 'dummyorg-dummyapp-commits.json'), 'r'))
    logger.info("Commits data sent")
    return jsonify(response), 200


@app.route('/orgs/dummyorg/repos', methods=['GET'])
def get_dummy_org_repos():
    response = json.load(open(os.path.join(templates_dir, 'dummyorg-repositories.json'), 'r'))
    logger.info("Special organization repositories data sent")
    return jsonify(response), 200


@app.route('/orgs/specialcaseorg/repos', methods=['GET'])
def get_special_org_repos():
    response = json.load(open(os.path.join(templates_dir, 'specialcase-repositories.json'), 'r'))
    logger.info("Special organization repositories data sent")
    return jsonify(response), 200


@app.route('/orgs/<org_name>/repos', methods=['GET'])
def get_combined_org_repos(org_name):
    if org_name not in ["dummyorg", "stackql"]:
        logger.warning(f"Invalid organization name: {org_name}")
        return jsonify({"error": "Invalid organization name"}), 400
    logger.info(f"Repositories data sent for organization: {org_name}")
    repositories = json.load(open(os.path.join(templates_dir, 'repositories.json'), 'r'))
    return jsonify(repositories), 200


@app.route('/repos/<string:org>/<string:repositoryId>/collaborators', methods=['GET'])
def get_collaborators(org, repositoryId):
    """Retrieve collaborators for a specific repository."""
    collaborators = json.load(open(os.path.join(templates_dir, 'collaborators.json'), 'r'))
    return jsonify(collaborators), 200


if __name__ == '__main__':
    app.run(debug=True)
