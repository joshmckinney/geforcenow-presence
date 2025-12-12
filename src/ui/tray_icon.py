import logging
import threading
import time
from PyQt5.QtWidgets import QSystemTrayIcon, QMenu, QAction, QApplication, QMessageBox, QProgressDialog
from PyQt5.QtGui import QIcon
from PyQt5.QtCore import Qt
from PyQt5.QtWidgets import QDialog
from src.core.utils import ASSETS_DIR, LOG_FILE
from src.core.app_launcher import AppLauncher
from src.ui.dialogs import AskGameDialog, MatchSelectionDialog, GamingMessageBox, GamingInputDialog, GAMING_STYLESHEET
from src.core.utils import get_lang_from_registry, load_locale

try:
    LANG = get_lang_from_registry()
    TEXTS = load_locale(LANG)
except Exception:
    LANG = os.getenv('GEFORCE_LANG', 'en')
    TEXTS = load_locale(LANG)

logger = logging.getLogger('geforce_presence')

class SystemTrayIcon(QSystemTrayIcon):
    def __init__(self, presence_manager, texts, parent=None):
        super().__init__(parent)
        self.pm = presence_manager
        TEXTS = texts
        
        self.setIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setToolTip("GeForce NOW Presence")
        
        self.menu = QMenu(parent)
        
        # Apply dark theme / advanced visual stylesheet
        self.menu.setStyleSheet("""
            QMenu {
                background-color: #1e1f22; /* Discord-like dark background */
                color: #dcddde;            /* Light gray text */
                border: 1px solid #111111;
                border-radius: 8px;
                padding: 5px;
            }
            QMenu::item {
                background-color: transparent;
                padding: 8px 24px 8px 12px;
                border-radius: 4px;
                margin: 2px 4px;
            }
            QMenu::item:selected {
                background-color: #045D0E; /* Discord Blurple */
                color: white;
            }
            QMenu::separator {
                height: 1px;
                background: #3f4145;
                margin: 6px 8px;
            }
        """)

        self.create_menu()
        self.setContextMenu(self.menu)
        
        # Connect signals
        self.pm.request_match_selection.connect(self.on_match_selection_requested)
        self.activated.connect(self.on_activated)
        self.menu.aboutToShow.connect(self.update_menu)

    def create_menu(self):
        self.menu.clear()
        
        # Force Game
        force_text = TEXTS.get("tray_force_game", "Force game...")
        if self.pm.forced_game:
            game_name = self.pm.forced_game.get('name', 'Unknown')
            if len(game_name) > 20:
                game_name = game_name[:17] + "..."
            force_text = f"Stop forcing: {game_name}"
            
        force_action = QAction(force_text, self.menu)
        force_action.triggered.connect(self.toggle_force_game)
        self.menu.addAction(force_action)
        
        # Obtain Cookie
        cookie_action = QAction(TEXTS.get("tray_get_cookie", "Obtain Steam cookie"), self.menu)
        cookie_action.triggered.connect(self.obtain_cookie)
        self.menu.addAction(cookie_action)
        
        # Open GeForce
        open_gf_action = QAction(TEXTS.get("tray_open_geforce", "Open GeForce NOW"), self.menu)
        open_gf_action.triggered.connect(self.open_geforce)
        self.menu.addAction(open_gf_action)
        
        # Set Max Party Size
        DEFAULT_CLIENT_ID = "1095416975028650046"
        current_cid = getattr(self.pm, "_connected_client_id", None)
        
        if current_cid and current_cid != DEFAULT_CLIENT_ID:
            party_action = QAction(TEXTS.get("tray_set_max_party_size", "Set Max Party Size..."), self.menu)
            party_action.triggered.connect(self.set_max_party_size_dialog)
            self.menu.addAction(party_action)

        # Sync Games
        sync_text = TEXTS.get("tray_sync_games", "Sync games")
        sync_action = QAction(sync_text, self.menu)
        sync_action.triggered.connect(self.sync_games)
        self.menu.addAction(sync_action)
        
        # Open Logs
        logs_action = QAction(TEXTS.get("tray_open_logs", "Open logs"), self.menu)
        logs_action.triggered.connect(self.open_logs)
        self.menu.addAction(logs_action)
        
        self.menu.addSeparator()
        
        # Exit
        exit_action = QAction(TEXTS.get("tray_exit", "Exit"), self.menu)
        exit_action.triggered.connect(QApplication.instance().quit)
        self.menu.addAction(exit_action)

    def update_menu(self):
        self.create_menu()

    def on_activated(self, reason):
        if reason == QSystemTrayIcon.DoubleClick:
            self.open_geforce()

    def toggle_force_game(self):
        if self.pm.forced_game:
            self.pm.stop_force_game()
            self.showMessage("OK", "Forzado de juego detenido.", QSystemTrayIcon.Information, 3000)
            self.update_menu()
            return

        dialog = AskGameDialog(title=TEXTS.get("force_game", "Force Game"), message=TEXTS.get("game_name", "Game Name:"))
        if dialog.exec_() == QDialog.Accepted:
            game_name = dialog.get_game_name()
            if not game_name:
                return
            
            self.process_force_game(game_name)

    def process_force_game(self, game_name):
        gm = self.pm.games_map or {}
        candidates = [k for k in gm if game_name.lower() in k.lower()]
        
        options = []
        if candidates:
            for k in candidates:
                options.append({"name": k, "id": gm[k].get("client_id"), "exe": gm[k].get("executable_path"), "score": 1.0})
        else:
            # 1. Search in Discord (Local Cache First)
            options = self.pm._find_discord_matches(game_name, max_candidates=5)

            # 2. If no matches, force download and search again
            if not options:
                self.showMessage("Buscando...", f"No encontrado en caché. Descargando datos recientes de Discord para '{game_name}'...", QSystemTrayIcon.Information, 4000)
                QApplication.processEvents() # Keep UI responsive (mostly)
                
                # Update cache
                self.pm._fetch_discord_apps_cached(force_download=True)
                
                # Search again
                options = self.pm._find_discord_matches(game_name, max_candidates=5)

            # Note: We don't apply automatically here loop; we show selection dialog

        if not options:
            self.showMessage("Info", "Sin coincidencias en JSON ni Discord (incluso tras actualizar).", QSystemTrayIcon.Information, 3000)
            return

        # Show selection dialog
        sel_dialog = MatchSelectionDialog("Seleccionar juego", options)
        if sel_dialog.exec_() == QDialog.Accepted and sel_dialog.selected_match:
            match = sel_dialog.selected_match
            self.apply_force_game(match)

    def apply_force_game(self, match):
        name = match["name"]
        cid = match.get("id")
        exe = match.get("exe")
        
        # PERSISTENCE: Save the match to games_config_merged.json
        # This ensures next time we have it in games_map and don't need to search/download
        self.pm._apply_discord_match(name, match)
        
        if cid:
            try:
                def reconnect_after_delay():
                    time.sleep(11)
                
                self.pm._disconnect_rpc_temporarily()
                
                self.pm.client_id = cid
                self.pm._connect_rpc(cid)
                logger.info(f"🔁 RPC reconectado con client_id forzado: {cid}")
            except Exception as e:
                logger.error(f"❌ Error reconectando RPC tras forzar juego: {e}")
                threading.Thread(target=reconnect_after_delay, daemon=True).start()

        if exe:
            try:
                self.pm.close_fake_executable()
            except Exception as e:
                logger.debug(f"No se pudo cerrar ejecutable previo: {e}")
            self.pm.launch_fake_executable(exe)

        self.pm.forced_game = {
            "name": name,
            "client_id": cid,
            "executable_path": exe
        }
        self.pm.last_game = dict(self.pm.forced_game)
        logger.info(f"🎮 Juego forzado activado: {name} (id={cid})")
        
        self.showMessage("OK", f"{TEXTS.get('tray_forced_game', 'Forced game')}: {name}", QSystemTrayIcon.Information, 3000)
        self.update_menu()

    def obtain_cookie(self):
        def confirm_callback(title, message):
            return GamingMessageBox.show_question(None, title, message)

        cookie = self.pm.cookie_manager.ask_and_obtain_cookie(confirm_callback)
        if cookie:
            self.pm.update_cookie(cookie)
            self.showMessage(TEXTS.get("cookie_title", "Cookie"), TEXTS.get("cookie_saved", "Cookie saved"), QSystemTrayIcon.Information, 3000)
        else:
            self.showMessage(TEXTS.get("cookie_title", "Cookie"), TEXTS.get("cookie_invalid", "Cookie invalid"), QSystemTrayIcon.Warning, 3000)

    def open_geforce(self):
        AppLauncher.launch_geforce_now()

    def open_logs(self):
        import os
        if LOG_FILE.exists():
            os.startfile(LOG_FILE)
        else:
            self.showMessage(TEXTS.get("logs_title", "Logs"), TEXTS.get("open_logs_error", "No log file found."), QSystemTrayIcon.Warning, 3000)

    def set_max_party_size_dialog(self):
        from PyQt5.QtWidgets import QInputDialog
        
        if not self.pm.last_game and not self.pm.forced_game:
            GamingMessageBox.show_warning(None, "Party Size", "No hay ningún juego en ejecución (ni forzado).")
            return

        current_max = 4
        
        # Try to get current values
        game = self.pm.forced_game or self.pm.last_game
        if game:
            game_key = game.get("name")
            if game_key and game_key in self.pm.games_map:
                existing = self.pm.games_map[game_key].get("max_party_size")
                if existing:
                    current_max = int(existing)

        i, ok = GamingInputDialog.get_int(None, "Set Party Size", 
                                    f"Tamaño MÁXIMO del grupo:", 
                                    current_max, 1, 100, 1)
        if ok:
            success = self.pm.set_max_party_size(i)
            if success:
                self.showMessage("Party Size", f"Tamaño máximo actualizado a {i}", QSystemTrayIcon.Information, 2000)
            else:
                self.showMessage("Error", "No se pudo actualizar el tamaño del grupo.", QSystemTrayIcon.Warning, 3000)

    def on_match_selection_requested(self, game_key, candidates):
        # This is called from PresenceManager when it finds a new game and needs user input
        # We need to run this in the main thread (which signals do automatically)
        dialog = MatchSelectionDialog(game_key, candidates)
        if dialog.exec_() == QDialog.Accepted:
            self.pm.on_match_selected(game_key, dialog.selected_match)
        else:
            self.pm.on_match_selected(game_key, None)

    def sync_games(self):
        status = self.pm.check_discord_cache_status()
        force = False
        
        if status["status"] == "FRESH":
            hours = status["hours"]
            msg = f"El archivo de caché se actualizó hace {hours:.1f} horas.\n¿Desea actualizarlo nuevamente?"
            if GamingMessageBox.show_question(None, "Sincronizar Juegos", msg):
                force = True
            # If No, we proceed with force=False (just local matching)
        
        # Create Progress Dialog
        self.progress = QProgressDialog("Sincronizando juegos...", "Cancelar", 0, 100, None)
        self.progress.setStyleSheet(GAMING_STYLESHEET)
        self.progress.setWindowModality(Qt.WindowModal)
        self.progress.setMinimumDuration(0)
        self.progress.setValue(0)
        self.progress.canceled.connect(self.on_sync_canceled)
        self.progress.show()
        
        # Connect signals
        try:
            self.pm.sync_progress.disconnect()
            self.pm.sync_finished.disconnect()
            self.pm.sync_error.disconnect()
        except:
            pass

        self.pm.sync_progress.connect(self.on_sync_progress)
        self.pm.sync_finished.connect(self.on_sync_finished)
        self.pm.sync_error.connect(self.on_sync_error)
        
        # Start thread
        threading.Thread(target=self.pm.sync_missing_game_details, args=(force,), daemon=True).start()

    def on_sync_canceled(self):
        logger.info("Solicitando cancelación de sincronización...")
        self.pm.cancel_sync()

    def on_sync_progress(self, current, total):
        if getattr(self, 'progress', None):
            self.progress.setMaximum(total)
            self.progress.setValue(current)

    def on_sync_finished(self, updated, total):
        if getattr(self, 'progress', None):
            self.progress.close()
            self.progress = None
        
        GamingMessageBox.show_info(None, "Sincronización Completada", f"Se han actualizado {updated} juegos de un total de {total} procesados.")
        
    def on_sync_error(self, error_msg):
        if getattr(self, 'progress', None):
            self.progress.close()
            self.progress = None
        GamingMessageBox.show_warning(None, "Error de Sincronización", f"Ocurrió un error: {error_msg}")
