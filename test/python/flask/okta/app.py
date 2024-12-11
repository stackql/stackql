from flask import Flask, jsonify, render_template, request
import os

app = Flask(__name__)

# Configure the templates directory
app.template_folder = os.path.join(os.path.dirname(__file__), "templates")

@app.route('/api/v1/users', methods=['GET'])
def get_users():
    """Handles paginated user responses."""
    after = int(request.args.get('after', 0))
    items_per_page = 10
    next_page = after + items_per_page
    max_items = 50  # Assuming a maximum of 100 users for demonstration

    users = []
    for i in range(after + 1, min(after + items_per_page + 1, max_items + 1)):
        users.append({
            "id": str(i),
            "status": "ACTIVE",
            "created": "2022-07-17T11:36:23.000Z",
            "activated": "2022-07-17T11:36:23.000Z",
            "statusChanged": "2022-07-17T11:37:06.000Z",
            "lastLogin": "2022-07-17T11:38:36.000Z",
            "lastUpdated": "2022-07-17T11:37:06.000Z",
            "passwordChanged": "2022-07-17T11:37:06.000Z",
            "type": {"id": "oty1l8curaosSnZLn697"},
            "profile": {
                "firstName": "sherlock",
                "lastName": "holmes",
                "mobilePhone": None,
                "secondEmail": None,
                "login": f"sherlockholmes{i}@gmail.com",
                "email": f"sherlockholmes{i}@gmail.com"
            },
            "credentials": {
                "password": {},
                "provider": {"type": "OKTA", "name": "OKTA"}
            },
            "_links": {
                "self": {
                    "href": f"https://dummyorg.okta.com/api/v1/users/{i}"
                }
            }
        })

    next_link = None
    if next_page <= max_items:
        next_link = f'<https://{request.host}/api/v1/users?after={next_page}>; rel="next"'

    headers = {
        'Datetime': 'now',
        'Link': next_link or f'<https://{request.host}/api/v1/users?after={after}>; rel="self"'
    }

    # response = {
    #     "users": users,
    #     "nextPage": next_link if next_link else None
    # }

    return jsonify(users), 200, headers

@app.route('/api/v1/apps', methods=['GET'])
def get_apps():
    """Returns app data."""
    return render_template('apps_template.json'), 200, {'Content-Type': 'application/json'}

if __name__ == '__main__':
    app.run(debug=True)
