import os
import time
import logging
import requests
import psutil
from pathlib import Path
from typing import Optional, Callable, Dict

try:
    import browser_cookie3
except ImportError:
    browser_cookie3 = None

from selenium import webdriver
from selenium.webdriver.edge.service import Service as EdgeService
from selenium.webdriver.edge.options import Options
from selenium.common.exceptions import WebDriverException

from src.core.utils import save_cookie_to_env, DRIVER_PATH, ensure_driver_executable, ENV_PATH

logger = logging.getLogger('geforce_presence')

class CookieManager:
    def __init__(self, texts: Dict, env_cookie: Optional[str] = None, test_url: str = ""):
        self.texts = texts
        self.env_cookie = env_cookie
        self.test_url = test_url
        self.driver_path = str(ensure_driver_executable(DRIVER_PATH))

    def validar_cookie(self, cookie_value: str) -> bool:
        try:
            s = requests.Session()
            s.cookies.set('steamLoginSecure', cookie_value, domain='steamcommunity.com')
            r = s.get(self.test_url, timeout=10)
            if r.status_code == 200 and "Sign In" not in r.text and "login" not in r.url.lower():
                return True
        except Exception as e:
            logger.debug(f"Error validando cookie: {e}")
        return False

    def get_cookie_from_edge_profile(self) -> Optional[str]:
        if not browser_cookie3:
            logger.warning("browser_cookie3 no instalado.")
            return None
            
        try:
            logger.info("🧩 Intentando leer cookie de Steam desde Edge (browser_cookie3)...")
            cj = browser_cookie3.edge(domain_name='steamcommunity.com')
            for cookie in cj:
                if cookie.name == 'steamLoginSecure':
                    logger.info("✅ Cookie automática obtenida desde Edge (browser_cookie3).")
                    return cookie.value
            logger.info("⚠️ No se encontró cookie steamLoginSecure en perfiles accesibles por browser_cookie3.")
        except Exception as e:
            logger.debug(f"browser_cookie3 fallo: {e}")
        return None
    
    def close_edge_processes(self):
        """Cierra todos los procesos de Microsoft Edge."""
        closed = 0
        for proc in psutil.process_iter(['pid', 'name']):
            try:
                if proc.info['name'] and "msedge.exe" in proc.info['name'].lower():
                    proc.terminate()
                    closed += 1
            except Exception:
                continue
        if closed:
            logger.info(f"🔒 {closed} procesos de Edge terminados.")
        else:
            logger.debug("No había procesos de Edge en ejecución.")

    def get_cookie_with_selenium(self, 
                                 headless: bool = False, 
                                 profile_dir: str = "Default", 
                                 confirm_callback: Optional[Callable[[str, str], bool]] = None) -> Optional[str]:
        try:
            # Check if Edge is running
            edge_running = any(
                (p.info['name'] and "msedge.exe" in p.info['name'].lower())
                for p in psutil.process_iter(['name'])
            )

            if edge_running:
                if confirm_callback:
                    res = confirm_callback(
                        self.texts.get("edge_open", "Microsoft Edge está abierto"), 
                        self.texts.get('edge_open_confirm', 'Edge needs to be closed to proceed. Close it?')
                    )
                    if not res:
                        logger.info("⏭️ Usuario canceló la obtención de cookie porque Edge estaba abierto.")
                        return None
                else:
                    logger.info("Edge is running and no callback provided to confirm close.")
                    return None

                self.close_edge_processes()
                time.sleep(2)

            logger.info("🧩 Obteniendo cookie de Steam con Selenium (Edge)...")
            
            localapp = os.getenv("LOCALAPPDATA", "")
            user_data_dir = str(Path(localapp) / "Microsoft" / "Edge" / "User Data")
            if not Path(user_data_dir).exists():
                logger.error("❌ No se encontró la carpeta de perfiles de Edge.")
                return None

            service = EdgeService(executable_path=self.driver_path)
            options = Options()
            options.add_argument(f"--user-data-dir={user_data_dir}")
            options.add_argument(f"--profile-directory={profile_dir}")
            if headless:
                options.add_argument("--headless=new")

            driver = webdriver.Edge(service=service, options=options)
            try:
                driver.get("https://steamcommunity.com")
                cookies = driver.get_cookies()
                for c in cookies:
                    if c.get('name') == 'steamLoginSecure':
                        val = c.get('value')
                        save_cookie_to_env(val, ENV_PATH)
                        logger.debug(f"Cookie obtenida parcial: {val[:20]}... (longitud: {len(val)})")
                        logger.info("✅ Cookie obtenida con Selenium.")
                        return val
                logger.warning("⚠️ No se encontró 'steamLoginSecure' en la sesión de Steam.")
            finally:
                driver.quit()
                
        except WebDriverException as e:
            msg = getattr(e, "msg", str(e))
            logger.error(f"❌ Selenium WebDriver error: {msg}")

            # Detecta exactamente el error de versión
            if "only supports Microsoft Edge version" or "Unable to obtain driver for MicrosoftEdge" in msg:
                logger.warning("🔄 Edge WebDriver desactualizado. Intentando actualizar...")

                try:
                    from src.core.edge_updater import EdgeDriverUpdater
                    driver_updater = EdgeDriverUpdater(parent_widget=None)
                    driver_updater.update()
                    logger.info("🆗 WebDriver actualizado correctamente. Reintentando Selenium...")

                    # Reintentar UNA sola vez
                    return self.get_cookie_with_selenium(
                        headless=headless,
                        profile_dir=profile_dir,
                        confirm_callback=confirm_callback
                    )

                except Exception as update_error:
                    logger.error(f"❌ Error actualizando Edge WebDriver: {update_error}")

            else:
                logger.error("⚠️ Error de Selenium no relacionado con el driver desactualizado.")
        except Exception as e:
            logger.error(f"⚠️ Error inesperado obteniendo cookie con Selenium: {e}")
            return None

    def get_steam_cookie(self, confirm_callback: Optional[Callable[[str, str], bool]] = None) -> Optional[str]:
        if self.env_cookie:
            logger.info("🧩 Validando cookie desde .env...")
            if self.validar_cookie(self.env_cookie):
                logger.info("✅ Cookie del .env válida.")
                return self.env_cookie
            else:
                logger.warning("⚠️ Cookie del .env expirada o inválida.")

        c = self.get_cookie_from_edge_profile()
        if c and self.validar_cookie(c):
            return c

        # If we are here, we need to ask user permission to use Selenium if not headless/silent
        if confirm_callback:
             if not confirm_callback("Cookie", self.texts.get('ask_cookie', 'Obtain cookie via Edge?')):
                 return None

        c2 = self.get_cookie_with_selenium(headless=False, confirm_callback=confirm_callback)
        if c2 and self.validar_cookie(c2):
            return c2

        logger.error("❌ No se pudo obtener cookie de Steam automáticamente.")
        return None

    def ask_and_obtain_cookie(self, confirm_callback: Callable[[str, str], bool]) -> Optional[str]:
        """Versión interactiva"""
        try:
            should = confirm_callback("Cookie", 
                                self.texts.get('ask_cookie', 'The program will try to obtain your Steam cookie using Microsoft Edge. Make sure you are logged in to Steam in Edge.\n\nDo you want to continue?'))

            if not should:
                logger.info("No se obtuvo cookie de Steam de forma interactiva.")
                return None

            c2 = self.get_cookie_with_selenium(headless=False, confirm_callback=confirm_callback)
            if c2 and self.validar_cookie(c2):
                # save_cookie_to_env is called inside get_cookie_with_selenium if successful? 
                # Actually I put it there.
                return c2
            
            c = self.get_cookie_from_edge_profile()
            if c and self.validar_cookie(c):
                # save_cookie_to_env(c) # Should save if found
                return c

            logger.warning("No se pudo obtener cookie automáticamente tras solicitud del usuario.")
            return None
            
        except Exception as e:
            logger.error(f"Error en ask_and_obtain_cookie: {e}")
            return None
