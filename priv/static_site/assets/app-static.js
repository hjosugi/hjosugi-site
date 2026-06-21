(() => {
  const body = document.body;
  const toggle = document.getElementById("crt-toggle");
  const stored = localStorage.getItem("hjosugi-hub-crt");
  if (stored === "off") {
    body.classList.add("crt-off");
    if (toggle) {
      toggle.textContent = "crt:off";
      toggle.setAttribute("aria-pressed", "false");
    }
  }

  toggle?.addEventListener("click", () => {
    const off = body.classList.toggle("crt-off");
    localStorage.setItem("hjosugi-hub-crt", off ? "off" : "on");
    toggle.textContent = off ? "crt:off" : "crt:on";
    toggle.setAttribute("aria-pressed", String(!off));
  });

  document.addEventListener("keydown", (event) => {
    const target = event.target;
    const typing = target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement;
    if (event.key === "/" && !typing) {
      const search = document.getElementById("radar-search");
      if (search) {
        event.preventDefault();
        search.focus();
      }
    }
  });
})();
