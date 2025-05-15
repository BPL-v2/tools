import json
import time
import requests
import os
import dotenv
dotenv.load_dotenv()


def get_guild_join_requests():
    cookies = {'POESESSID': os.getenv('POESESSID')}

    headers = {
        'accept': 'application/json',
        'user-agent': 'Liberatorist@gmail.com',
    }

    params = {
        'sort': 'roleDesc',
        'search': '',
        'offset': '0',
        'limit': '100',
        '_': '1747331123076',
    }
    response = requests.get(
        'https://www.pathofexile.com/api/private-league-member/65013',
        params=params,
        cookies=cookies,
        headers=headers,
    )
    return [member for member in response.json()["members"] if member["role"] == "requested_invite"]


def get_sorted_users():
    headers = {'Authorization': f'Bearer {os.getenv("BPL_TOKEN")}'}
    response = requests.get(
        f'https://v2202503259898322516.goodsrv.de/api/events/current/signups', headers=headers)
    return {player["user"]["account_name"] for player in response.json() if player["team_id"] is not None}


def accept_guild_invites(members):
    cookies = {
        'POESESSID': os.getenv('POESESSID'),
    }
    headers = {
        'content-type': 'application/json',
        'user-agent': 'Liberatorist@gmail.com',
    }
    json_data = [

        {
            'name': 'accept',
            'value': member["id"],
        }
        for member in members
    ]
    return requests.post(
        'https://www.pathofexile.com/api/private-league-member/65013',
        cookies=cookies,
        headers=headers,
        json=json_data,
    )


def handle_guild_invites():
    sorted_users = get_sorted_users()
    members_to_add = [member for member in get_guild_join_requests(
    ) if member["memberName"] in sorted_users and member["isAcceptable"]]
    unknown_users = [
        member for member in get_guild_join_requests() if member["memberName"] not in sorted_users]
    if unknown_users:
        print(
            f"Unknown users requesting invites: {json.dumps(unknown_users, indent=0)}")
    response = accept_guild_invites(members_to_add)
    if response.status_code == 200:
        print(f"{len(members_to_add)} Invites accepted successfully.")
    else:
        print(f"Failed to accept invites. Status code: {response.status_code}")
        print(response.text)


if __name__ == "__main__":
    while True:
        print("Checking for guild invites...")
        handle_guild_invites()
        time.sleep(300)
