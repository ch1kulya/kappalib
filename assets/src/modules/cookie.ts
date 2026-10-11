export function initCookieNotice(): void {
  const notice = document.getElementById("cookie-notice");
  const acceptBtn = document.getElementById("cookie-notice-accept");

  if (!notice || !acceptBtn) {
    console.warn("Cookie notice elements missing in DOM");
    return;
  }

  const isConfirmed = localStorage.getItem("cookieConsent") === "true";
  if (!isConfirmed) {
    console.info("Cookie notice required, showing banner");
    notice.style.display = "";
  } else {
    console.info("Cookie notice already confirmed");
  }

  acceptBtn.onclick = () => {
    console.info("Cookie notice confirmed");
    localStorage.setItem("cookieConsent", "true");
    notice.style.display = "none";
  };
}
