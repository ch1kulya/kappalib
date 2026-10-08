import { trackEvent } from "./analytics";

export function initCookieNotice(): void {
  const notice = document.getElementById("cookie-notice");
  const acceptBtn = document.getElementById("cookie-notice-accept");
  const declineBtn = document.getElementById("cookie-notice-decline");

  if (!notice || !acceptBtn || !declineBtn) {
    return;
  }

  const consent = localStorage.getItem("cookieConsent");
  if (!consent) {
    notice.style.display = "";
  }

  acceptBtn.onclick = () => {
    localStorage.setItem("cookieConsent", "accepted");
    trackEvent("cookie_consent_accepted");
    notice.style.display = "none";
  };

  declineBtn.onclick = () => {
    localStorage.setItem("cookieConsent", "declined");
    notice.style.display = "none";
  };
}
