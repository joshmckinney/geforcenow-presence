
import requests
import json

url = "https://discord.com/api/v9/applications/detectable"
try:
    print(f"Fetching {url}...")
    resp = requests.get(url)
    resp.raise_for_status()
    data = resp.json()
    print(f"Total apps: {len(data)}")
    
    found = False
    for app in data:
        if "pragmata" in app.get("name", "").lower():
            print("FOUND PRAGMATA:")
            print(json.dumps(app, indent=4))
            found = True
            
    if not found:
        print("Pragmata NOT found in online list.")
        
except Exception as e:
    print(f"Error: {e}")
