import time
import requests
import os
import dotenv
dotenv.load_dotenv()


team_shorts = {
    18: "BAS",
    19: "DEA",
    20: "SNI"
}


def get_ladder():
    response = requests.get(
        f'{os.environ.get("BPL_BASE_URL")}/events/current/ladder')
    return response.json()


def get_users():
    response = requests.get(
        f'{os.environ.get("BPL_BASE_URL")}/events/current/users')

    userMap = {}
    for teamId, users in response.json().items():
        for user in users:
            userMap[user["id"]] = int(teamId)
    return userMap


def team_check():

    userMap = get_users()
    ladderEntries = get_ladder()

    for entry in ladderEntries:

        if entry["user_id"] in userMap:
            teamId = userMap[entry["user_id"]]
            team_short = team_shorts[teamId]
            if team_short.lower() not in entry["character_name"].lower():
                print(
                    f"Mismatch: {entry['character_name']} should have team name {team_short}")


if __name__ == "__main__":
    while True:
        print(time.strftime("%Y-%m-%d %H:%M:%S", time.localtime()),
              "Checking for player name mismatches...")
        team_check()
        time.sleep(300)
