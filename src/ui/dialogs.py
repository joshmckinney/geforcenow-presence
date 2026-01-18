from PyQt5.QtWidgets import (QDialog, QVBoxLayout, QLabel, QLineEdit, QPushButton, 
                             QListWidget, QHBoxLayout, QMessageBox, QWidget)
from PyQt5.QtGui import QIcon, QMovie
from PyQt5.QtCore import Qt, QSize
from src.core.utils import ASSETS_DIR
from src.core.utils import get_lang_from_registry, load_locale
import os
import logging
try:
    LANG = get_lang_from_registry()
    TEXTS = load_locale(LANG)
except Exception:
    LANG = os.getenv('GEFORCE_LANG', 'en')
    TEXTS = load_locale(LANG)

logger = logging.getLogger('geforce_presence')


# ---- ESTILOS GLOBALES ----
GAMING_STYLESHEET = """
    QDialog {
        background-color: #0d0e10;
        border: 2px solid #1b1f23;
        border-radius: 14px;
    }

    QLabel {
        font-size: 14px;
        font-family: "Segoe UI";
        color: #e0e0e0;
        padding-bottom: 4px;
    }
    
    QLabel#title_label {
        font-size: 18px;
        font-weight: bold;
        color: #ffffff;
        padding-bottom: 8px;
    }

    QLineEdit, QSpinBox {
        padding: 8px;
        font-size: 14px;
        border: 1px solid #2c2f33;
        border-radius: 6px;
        background: #1a1b1d;
        color: #ffffff;
        font-family: "Segoe UI";
        font-weight: bold;
    }

    QLineEdit:focus, QSpinBox:focus {
        border: 2px solid #454C55;
    }

    QPushButton {
        background-color: #045D0E;
        color: #FFFFFF;
        padding: 8px 16px;
        border-radius: 6px;
        font-size: 14px;
        font-family: "Segoe UI";
        font-weight: bold;
    }

    QPushButton:hover {
        background-color: #12881F;
    }
    
    QPushButton:pressed {
        background-color: #03420a;
    }

    QPushButton#secondary {
        background-color: #2c2f33;
        color: #e6e6e6;
    }

    QPushButton#secondary:hover {
        background-color: #3c3f43;
    }

    /* LIST WIDGET & SCROLLBARS */
    QListWidget {
        background: #131416;
        border: 1px solid #1f2428;
        border-radius: 8px;
        padding: 6px;
        font-size: 13px;
        font-family: Consolas, monospace;
        color: #cfcfcf;
    }

    QListWidget::item {
        padding: 8px;
        border-radius: 4px;
        color: #dfdfdf;
    }

    QListWidget::item:selected {
        background-color: #00e676;
        color: #0e0f11;
        font-weight: bold;
    }

    QScrollBar:vertical {
        background: transparent;
        width: 8px;
        margin: 4px 0;
    }
    QScrollBar::handle:vertical {
        background: #383a3d;
        border-radius: 4px;
        min-height: 30px;
    }
    QScrollBar::handle:vertical:hover {
        background: #4a4d50;
    }
    QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical {
        height: 0; 
        background: none; 
    }
"""

class GamingMessageBox(QDialog):
    def __init__(self, title, text, icon_type="info", parent=None):
        super().__init__(parent)
        self.setWindowTitle(title)
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)
        self.setStyleSheet(GAMING_STYLESHEET)
        
        # Layout
        layout = QVBoxLayout()
        layout.setContentsMargins(25, 25, 25, 20)
        layout.setSpacing(20)
        
        # Icon & Text Row (Optional: add an icon label if desired, skipping for simplicity to match style)
        self.lbl_text = QLabel(text)
        self.lbl_text.setWordWrap(True)
        self.lbl_text.setAlignment(Qt.AlignCenter)
        self.lbl_text.setStyleSheet("font-size: 15px;")
        layout.addWidget(self.lbl_text)
        
        # Buttons
        btn_layout = QHBoxLayout()
        btn_layout.setSpacing(15)
        
        self.ok_btn = QPushButton("OK")
        self.ok_btn.clicked.connect(self.accept)
        
        self.cancel_btn = QPushButton("Cancel")
        self.cancel_btn.setObjectName("secondary")
        self.cancel_btn.clicked.connect(self.reject)
        
        if icon_type == "question":
            self.ok_btn.setText(TEXTS.get("yes", "Yes"))
            self.cancel_btn.setText(TEXTS.get("no", "No"))
            btn_layout.addWidget(self.ok_btn)
            btn_layout.addWidget(self.cancel_btn)
        else:
            # Info / Warning
            btn_layout.addStretch()
            btn_layout.addWidget(self.ok_btn)
            btn_layout.addStretch()
            
        layout.addLayout(btn_layout)
        self.setLayout(layout)
        # Auto size
        self.adjustSize()

    @staticmethod
    def show_info(parent, title, text):
        dlg = GamingMessageBox(title, text, "info", parent)
        dlg.exec_()
        
    @staticmethod
    def show_warning(parent, title, text):
        dlg = GamingMessageBox(title, text, "warning", parent)
        dlg.exec_()

    @staticmethod
    def show_question(parent, title, text):
        dlg = GamingMessageBox(title, text, "question", parent)
        return dlg.exec_() == QDialog.Accepted

class GamingInputDialog(QDialog):
    def __init__(self, title, label_text, value=0, min_val=0, max_val=100, step=1, parent=None):
        super().__init__(parent)
        self.setWindowTitle(title)
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)
        self.setStyleSheet(GAMING_STYLESHEET)
        
        layout = QVBoxLayout()
        layout.setContentsMargins(25, 25, 25, 20)
        layout.setSpacing(15)
        
        lbl = QLabel(label_text)
        lbl.setAlignment(Qt.AlignCenter)
        layout.addWidget(lbl)
        
        from PyQt5.QtWidgets import QSpinBox
        self.spin = QSpinBox()
        self.spin.setRange(min_val, max_val)
        self.spin.setValue(value)
        self.spin.setSingleStep(step)
        self.spin.setAlignment(Qt.AlignCenter)
        layout.addWidget(self.spin)
        
        btn_layout = QHBoxLayout()
        self.ok_btn = QPushButton("OK")
        self.ok_btn.clicked.connect(self.accept)
        self.cancel_btn = QPushButton("Cancel")
        self.cancel_btn.setObjectName("secondary")
        self.cancel_btn.clicked.connect(self.reject)
        
        btn_layout.addWidget(self.ok_btn)
        btn_layout.addWidget(self.cancel_btn)
        layout.addLayout(btn_layout)
        
        self.setLayout(layout)
        self.setFixedSize(300, 180)

    @staticmethod
    def get_int(parent, title, label, value=0, min_val=0, max_val=100, step=1):
        dlg = GamingInputDialog(title, label, value, min_val, max_val, step, parent)
        if dlg.exec_() == QDialog.Accepted:
            return dlg.spin.value(), True
        return value, False


class AskGameDialog(QDialog):
    def __init__(self, parent=None, title=TEXTS.get("force_game", "Force Game"),
                 message=TEXTS.get("game_name", "GAME NAME:")):
        super().__init__(parent)

        self.setWindowTitle(title)
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setFixedSize(420, 240) # Increased height for checkbox
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)

        # ---- 🎮 ESTILO GAMING OSCURO ----
        self.setStyleSheet(GAMING_STYLESHEET)

        # ---- LAYOUT ----
        layout = QVBoxLayout()
        layout.setContentsMargins(25, 25, 25, 15)
        layout.setSpacing(15)

        # Centrado del label
        self.label = QLabel(message)
        self.label.setObjectName("title_label")
        self.label.setAlignment(Qt.AlignCenter)  
        layout.addWidget(self.label)

        self.entry = QLineEdit()
        layout.addWidget(self.entry)

        # Checkbox for Quest Mode
        from PyQt5.QtWidgets import QCheckBox
        self.quest_mode_cb = QCheckBox(TEXTS.get("quest_mode", "Discord Quest Mode (Multiple Games)"))
        self.quest_mode_cb.setStyleSheet("color: #e0e0e0; font-size: 13px; font-weight: bold;")
        layout.addWidget(self.quest_mode_cb)

        # Botones más compactos
        btn_layout = QHBoxLayout()
        btn_layout.setSpacing(12)

        self.ok_btn = QPushButton(TEXTS.get("ok", "OK"))
        self.cancel_btn = QPushButton(TEXTS.get("cancel", "Cancel"))
        self.cancel_btn.setObjectName("secondary")

        self.ok_btn.clicked.connect(self.accept)
        self.cancel_btn.clicked.connect(self.reject)

        btn_layout.addWidget(self.ok_btn)
        btn_layout.addWidget(self.cancel_btn)

        layout.addLayout(btn_layout)
        self.setLayout(layout)

        # ---- 🎞️ ANIMATED BACKGROUND ----
        self.bg_label = QLabel(self)
        self.gif = QMovie(str(ASSETS_DIR / "nvidia.gif"))
        self.bg_label.setMovie(self.gif)
        self.bg_label.setScaledContents(True)
        self.gif.start()
        # Ensure it stays behind
        self.bg_label.lower()

    def resizeEvent(self, event):
        if hasattr(self, 'bg_label'):
            self.bg_label.resize(self.size())
        super().resizeEvent(event)

    def get_game_name(self):
        return self.entry.text()

    def is_quest_mode(self):
        return self.quest_mode_cb.isChecked()


class QuestListDialog(QDialog):
    def __init__(self, presence_manager, parent=None):
        super().__init__(parent)
        self.pm = presence_manager
        self.setWindowTitle("Discord Quest Mode - Active Games")
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setMinimumWidth(450)
        self.setMinimumHeight(400)
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)
        self.setStyleSheet(GAMING_STYLESHEET)
        
        layout = QVBoxLayout()
        layout.setContentsMargins(15, 15, 15, 15)
        layout.setSpacing(10)
        
        lbl = QLabel(TEXTS.get("active_games", "Juegos activos (15 minutos máx.)"))
        lbl.setObjectName("title_label")
        lbl.setAlignment(Qt.AlignCenter)
        layout.addWidget(lbl)
        
        self.list_widget = QListWidget()
        self.list_widget.setStyleSheet(GAMING_STYLESHEET + """
            QListWidget::item { 
                border-bottom: 1px solid #2c2f33; 
                margin-bottom: 4px;
            }
        """)
        layout.addWidget(self.list_widget)
        
        # Add new game button
        add_btn = QPushButton(TEXTS.get("force_new_game", "Forzar Nuevo Juego"))
        add_btn.clicked.connect(self.on_add_game)
        layout.addWidget(add_btn)
        
        self.close_btn = QPushButton(TEXTS.get("close_window", "Cerrar Ventana (Juegos continuarán)"))
        self.close_btn.setObjectName("secondary")
        self.close_btn.clicked.connect(self.accept)
        layout.addWidget(self.close_btn)
        
        self.setLayout(layout)
        
        # Timer for UI updates
        from PyQt5.QtCore import QTimer
        self.timer = QTimer(self)
        self.timer.timeout.connect(self.refresh_list)
        self.timer.start(1000) # Update every second
        
        self.refresh_list()
        
    def on_add_game(self):
        # Trigger the same logic as the tray icon
        # We can signal or call a callback provided in init, but for now let's assume parent/pm handling?
        # Ideally, we should invoke the main add game dialog.
        # But we are in a dialog.
        
        # Let's import AskGameDialog locally to avoid circulars if any, though we are in same file
        dlg = AskGameDialog(parent=self, message="Nombre del juego para Quest:")
        dlg.quest_mode_cb.setChecked(True)
        dlg.quest_mode_cb.setEnabled(False) # Force quest mode if adding from here
        
        if dlg.exec_() == QDialog.Accepted:
            game_name = dlg.get_game_name()
            if game_name:
                # We need to trigger the process_force_game logic.
                # Since we have `pm`, we can perhaps call a new method on it or use the tray icon logic?
                # The tray icon logic handles the searching/downloading.
                # We should probably expose that logic or signal it.
                # For simplicity, let's signal the PM to request a new quest game login.
                # But PM is core. Tray is UI.
                # Let's emit a custom signal if possible, or direct call if we move logic to PM.
                # For now, let's assume PM has a method `start_quest_game_flow(game_name, parent_ui)`
                # Or we can reuse the callback passed from tray?
                # Actually, the proper way is probably to emit a signal from this dialog that the Tray listens to?
                # But Tray creates this dialog.
                # We can call `self.parent().process_force_game(game_name, quest_mode=True)` if parent is tray.
                pass
                # To be handled in the connection logic in TrayIcon.
                # Actually, let's allow the user to type it here, but the heavy lifting is done by the caller.
                # We will define a callback.
                if hasattr(self, 'add_game_callback'):
                    self.add_game_callback(game_name)

    def set_add_game_callback(self, callback):
        self.add_game_callback = callback
        
    def refresh_list(self):
        # Save scroll position
        # current_row = self.list_widget.currentRow()
        
        self.list_widget.clear()
        
        quests = getattr(self.pm, "active_quests", {})
        if not quests:
            self.list_widget.addItem("No hay juegos activos en modo Quest.")
            return

        from PyQt5.QtWidgets import QWidget, QProgressBar, QHBoxLayout, QLabel, QPushButton
        
        sorted_quests = sorted(quests.items(), key=lambda x: x[1]['start_time'])
        
        for game_id, data in sorted_quests:
            item_widget = QWidget()
            layout = QVBoxLayout()
            layout.setContentsMargins(8, 8, 8, 8)
            layout.setSpacing(4)
            
            # Header
            header_layout = QHBoxLayout()
            name_lbl = QLabel(f"{data.get('name', 'Unknown')}")
            # Add padding and min-height to prevent clipping of descenders/ascenders
            name_lbl.setStyleSheet("color: #ffffff; font-size: 16px; font-weight: bold; padding: 2px 0px 4px 0px;")
            name_lbl.setWordWrap(True)
            # Ensure label tries to expand reasonably
            name_lbl.setMinimumHeight(24)
            header_layout.addWidget(name_lbl, 1) 
            
            # Close/Remove button
            btn_stop = QPushButton("❌")
            btn_stop.setFixedSize(28, 28)
            btn_stop.setCursor(Qt.PointingHandCursor)
            btn_stop.setStyleSheet("""
                QPushButton { background: #d32f2f; color: white; border: none; border-radius: 4px; font-size: 14px; }
                QPushButton:hover { background: #b71c1c; }
            """)
            btn_stop.clicked.connect(lambda checked, gid=game_id: self.stop_quest(gid))
            header_layout.addWidget(btn_stop)
            
            layout.addLayout(header_layout)
            
            # Spacer
            layout.addSpacing(4)

            # Progress status
            import time
            elapsed = time.time() - data['start_time']
            duration = 15 * 60 # 15 mins
            remaining = max(0, duration - elapsed)
            
            progress = QProgressBar()
            progress.setRange(0, duration)
            progress.setValue(int(remaining))
            progress.setTextVisible(False)
            
            # Color based on state or time
            progress.setStyleSheet("""
                QProgressBar {
                    background-color: #2c2f33;
                    border: none;
                    border-radius: 4px;
                    height: 10px;
                }
                QProgressBar::chunk {
                    background-color: #5865F2; /* Discord Blurple brighter */
                    border-radius: 4px;
                }
            """)
            
            if data.get('finished', False):
                status_text = "Estado: Detenido"
                progress.setValue(0)
            else:
                mins = int(remaining // 60)
                secs = int(remaining % 60)
                status_text = f"⏱️ Tiempo restante: {mins:02d}:{secs:02d}"
                
            status_lbl = QLabel(status_text)
            status_lbl.setStyleSheet("color: #dcddde; font-size: 13px; font-weight: 500; padding-top: 2px;")
            
            layout.addWidget(status_lbl)
            layout.addWidget(progress)
            
            item_widget.setLayout(layout)
            
            # Force layout calculation
            item_widget.adjustSize()
            
            # Add to list
            from PyQt5.QtWidgets import QListWidgetItem
            list_item = QListWidgetItem(self.list_widget)
            # Add a little extra height buffer to be safe
            sz = item_widget.sizeHint()
            sz.setHeight(sz.height() + 10) 
            list_item.setSizeHint(sz)
            self.list_widget.addItem(list_item)
            self.list_widget.setItemWidget(list_item, item_widget)
            
    def stop_quest(self, game_id):
        self.pm.stop_quest_game(game_id)
        self.refresh_list()



class MatchSelectionDialog(QDialog):
    def __init__(self, game_key, candidates, parent=None):
        super().__init__(parent)

        self.setWindowTitle(TEXTS.get("apply_discord_match", "Discord Match"))
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setMinimumWidth(540)
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)

        self.candidates = candidates
        self.selected_match = None

        # ---- 🎮 ESTILO GAMING OSCURO ----
        self.setStyleSheet(GAMING_STYLESHEET)

        # ---- LAYOUT ----
        layout = QVBoxLayout()
        layout.setContentsMargins(20, 20, 20, 15)
        layout.setSpacing(15)

        lbl = QLabel(
            TEXTS.get(
                "ask_discord_match",
                f"Se encontró un posible juego: '{game_key}'.\nSelecciona la coincidencia correcta:"
            )
        )
        layout.addWidget(lbl)

        self.list_widget = QListWidget()
        for c in candidates:
            exe = c.get("exe") or ""
            text = f"{c['name']}  ({c['score']:.2f})  [{exe}]  id={c.get('id') or '—'}"
            self.list_widget.addItem(text)

        layout.addWidget(self.list_widget)

        # ---- BOTONES ----
        btn_layout = QHBoxLayout()

        self.confirm_btn = QPushButton(TEXTS.get("confirm", "Confirmar"))
        self.confirm_btn.clicked.connect(self.on_confirm)

        self.ignore_btn = QPushButton(TEXTS.get("ignore", "Ignorar"))
        self.ignore_btn.setObjectName("secondary")
        self.ignore_btn.clicked.connect(self.reject)

        btn_layout.addWidget(self.confirm_btn)
        btn_layout.addWidget(self.ignore_btn)
        layout.addLayout(btn_layout)

        self.setLayout(layout)

    def on_confirm(self):
        row = self.list_widget.currentRow()
        if row >= 0:
            self.selected_match = self.candidates[row]
            self.accept()
        else:
            QMessageBox.warning(
                self,
                TEXTS.get("selection_required", "Selección requerida"),
                TEXTS.get("selection_required_msg", "Por favor selecciona una opción de la lista.")
            )


class CustomPresenceDialog(QDialog):
    def __init__(self, game_name, current_data, parent=None):
        super().__init__(parent)
        self.setWindowTitle(f"Custom Presence: {game_name}")
        self.setWindowIcon(QIcon(str(ASSETS_DIR / "geforce.ico")))
        self.setWindowFlags(self.windowFlags() | Qt.WindowStaysOnTopHint)
        self.setStyleSheet(GAMING_STYLESHEET)
        
        self.game_name = game_name
        self.result_data = None
        
        layout = QVBoxLayout()
        layout.setContentsMargins(20, 20, 20, 20)
        layout.setSpacing(15)
        
        # Helper to create rows
        def add_row(label_txt, widget):
            r = QVBoxLayout()
            r.setSpacing(5)
            l = QLabel(label_txt)
            r.addWidget(l)
            r.addWidget(widget)
            layout.addLayout(r)
            return widget

        self.details_edit = add_row("Detalles (Línea 1):", QLineEdit())
        self.details_edit.setPlaceholderText("Ej: Jugando Competitivo")
        self.details_edit.setText(current_data.get("custom_details", ""))

        self.state_edit = add_row("Estado (Línea 2):", QLineEdit())
        self.state_edit.setPlaceholderText("Ej: En grupo de 5")
        self.state_edit.setText(current_data.get("custom_state", ""))

        # Party Size Row
        party_layout = QHBoxLayout()
        
        from PyQt5.QtWidgets import QSpinBox
        self.party_current = QSpinBox()
        self.party_current.setRange(0, 100)
        self.party_current.setValue(current_data.get("custom_party_size_current", 0))
        
        self.party_max = QSpinBox()
        self.party_max.setRange(0, 100)
        self.party_max.setValue(current_data.get("custom_party_size_max", 0))
        
        p_sub = QVBoxLayout()
        p_sub.addWidget(QLabel("Personas (Actual):"))
        p_sub.addWidget(self.party_current)
        party_layout.addLayout(p_sub)
        
        p_sub2 = QVBoxLayout()
        p_sub2.addWidget(QLabel("Personas (Max):"))
        p_sub2.addWidget(self.party_max)
        party_layout.addLayout(p_sub2)
        
        layout.addLayout(party_layout)
        
        layout.addWidget(QLabel("Nota: Si 'Max' es 0, no se mostrará información de grupo."))

        # Buttons
        btn_layout = QHBoxLayout()
        self.save_btn = QPushButton("Guardar")
        self.save_btn.clicked.connect(self.on_save)
        
        self.cancel_btn = QPushButton("Cancelar")
        self.cancel_btn.setObjectName("secondary")
        self.cancel_btn.clicked.connect(self.reject)
        
        btn_layout.addWidget(self.save_btn)
        btn_layout.addWidget(self.cancel_btn)
        layout.addLayout(btn_layout)
        
        self.setLayout(layout)
        self.adjustSize()

    def on_save(self):
        self.result_data = {
            "custom_details": self.details_edit.text(),
            "custom_state": self.state_edit.text(),
            "custom_party_size_current": self.party_current.value(),
            "custom_party_size_max": self.party_max.value()
        }
        self.accept()
