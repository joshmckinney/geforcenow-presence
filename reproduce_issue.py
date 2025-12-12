
import sys
import logging
from src.core.presence_manager import PresenceManager
from src.core.utils import IS_WINDOWS

print(f"IS_WINDOWS: {IS_WINDOWS}")

# Script basico para obtener datos de prueba
app_data = {
    "id": "1448369915462549616",
    "name": "PRAGMATA",
    "aliases": [],
    "executables": [
        {
            "is_launcher": False, 
            "name": "pragmata_sketchbook.exe",
            "os": "win32"
        }
    ]
}

# Setup logging
logging.basicConfig(level=logging.DEBUG)

# Use the real PresenceManager now that we modified it
# We need to mock dependencies if we instantiate it fully, but we just want to test _add_candidate
# Ideally we instantiate a dummy that inherits from real one
class TestPresenceManager(PresenceManager):
    def __init__(self):
        # We just need to mock what _add_candidate uses.
        # It uses IS_WINDOWS (global), IS_MACOS (global), and self._add_candidate helper method which logic is inside.
        # It doesn't use self state other than for logging which is global 'logger'
        pass

pm = TestPresenceManager()
candidates = []
score = 1.0

try:
    pm._add_candidate(candidates, app_data, score)
    print("Candidates:", candidates)
    
    if candidates and candidates[0].get("exe") == "pragmata_sketchbook.exe":
        print("SUCCESS: Executable found.")
    else:
        print("FAILURE: Executable NOT found.")
        
except Exception as e:
    print(f"Error: {e}")
