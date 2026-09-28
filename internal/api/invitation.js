(() => {
  const russian = (navigator.language || "").toLowerCase().startsWith("ru");
  const text = russian ? {
    title: "Вас приглашают в Shum",
    description: "Нажмите кнопку, чтобы открыть профиль и добавить человека в контакты.",
    invalid: "Ссылка неполная или повреждена. Попросите отправителя поделиться контактом ещё раз.",
    open: "Открыть в Shum"
  } : {
    title: "An invitation to Shum",
    description: "Open the profile in Shum to add this person to your contacts.",
    invalid: "This link is incomplete or invalid. Ask the sender to share their contact again.",
    open: "Open in Shum"
  };
  if (!russian) {
    document.documentElement.lang = "en";
    document.title = "Shum contact";
    document.getElementById("install").textContent = "Install with TestFlight";
    document.getElementById("hint").textContent = "On your iPhone, install Shum and create a profile, then return to this page. If the button does not open the app, open this page in Safari.";
    document.getElementById("privacy").textContent = "This page does not request access to your conversations or add a contact without your confirmation in Shum.";
  }
  const update = () => {
    // Allow only bounded base64url payloads in known app routes. The app
    // decodes the card and verifies signatures before showing a profile.
    const legacy = /^#c2\/([A-Za-z0-9_-]{42}[AEIMQUYcgkosw048])$/.exec(window.location.hash);
    const profile = /^#(c4|c3|c)\/([A-Za-z0-9_-]{200,2048})$/.exec(window.location.hash);
    const target = legacy ? "shum://c2/" + legacy[1]
      : profile ? "shum://" + profile[1] + "/" + profile[2] : null;
    const open = document.getElementById("open");
    open.hidden = !target;
    open.removeAttribute("href");
    open.textContent = text.open;
    document.getElementById("title").textContent = text.title;
    document.getElementById("description").textContent = target ? text.description : text.invalid;
    if (target) open.href = target;
  };
  update();
  window.addEventListener("hashchange", update);
})();
