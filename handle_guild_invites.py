import time
import requests
import os
import dotenv
dotenv.load_dotenv()

event_id = 72796


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
        f'https://www.pathofexile.com/api/private-league-member/{event_id}',
        params=params,
        cookies=cookies,
        headers=headers,
    )
    return [member for member in response.json()["members"] if member["role"] == "requested_invite"]


def get_sorted_users():
    headers = {'Authorization': f'Bearer {os.getenv("BPL_TOKEN")}'}
    response = requests.get(
        f'{os.environ.get("BPL_BASE_URL")}/events/current/signups', headers=headers)
    return {player["user"]["account_name"] for player in response.json() if player["team_id"] is not None}


def accept_guild_invites(members):
    cookies = {
        'POESESSID': os.getenv('POESESSID'),
    }
    headers = {
        'accept': 'application/json, text/javascript, */*; q=0.01',
        'accept-language': 'en-US,en;q=0.9',
        'cache-control': 'no-cache',
        'content-type': 'application/json',
        'origin': 'https://www.pathofexile.com',
        'pragma': 'no-cache',
        'priority': 'u=1, i',
        'sec-ch-ua': '"Chromium";v="136", "Brave";v="136", "Not.A/Brand";v="99"',
        'sec-ch-ua-arch': '"x86"',
        'sec-ch-ua-bitness': '"64"',
        'sec-ch-ua-full-version-list': '"Chromium";v="136.0.0.0", "Brave";v="136.0.0.0", "Not.A/Brand";v="99.0.0.0"',
        'sec-ch-ua-mobile': '?0',
        'sec-ch-ua-model': '""',
        'sec-ch-ua-platform': '"Windows"',
        'sec-ch-ua-platform-version': '"15.0.0"',
        'sec-fetch-dest': 'empty',
        'sec-fetch-mode': 'cors',
        'sec-fetch-site': 'same-origin',
        'sec-gpc': '1',
        'user-agent': 'Liberatorist@gmail.com',
        'x-requested-with': 'XMLHttpRequest',
    }
    json_data = [

        {
            'name': 'accept',
            'value': member["id"],
        }
        for member in members
    ]
    return requests.post(
        f'https://www.pathofexile.com/api/private-league-member/{event_id}',
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
            f"Unknown users requesting invites: {', '.join([member['memberName'] for member in unknown_users])}")
    if not members_to_add:
        print("No new members to add.")
        return
    response = accept_guild_invites(members_to_add)
    if response.status_code == 200:
        print(f"{len(members_to_add)} Invites accepted successfully.")
    else:
        print(f"Failed to accept invites. Status code: {response.status_code}")
        print(response.text)


if __name__ == "__main__":
    while True:
        print(time.strftime("%Y-%m-%d %H:%M:%S", time.localtime()),
              "Checking for guild invites...")
        handle_guild_invites()
        time.sleep(300)
