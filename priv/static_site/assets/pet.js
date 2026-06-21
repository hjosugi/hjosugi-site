// Animates the 前方後円墳 (keyhole-tomb) mascot rendered by HjosugiHub.Kofun.
// The sprite markup lives in Elixir; this only makes it stroll and hop.
(() => {
  const pet = document.querySelector(".kofun-pet");
  if (!pet) return;
  const inner = pet.querySelector(".kofun-sprite");

  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const speed = 0.42; // px per ~16ms frame
  let x = 28;
  let dir = 1;
  let state = reduce ? "idle" : "walk";

  const clampMax = () => Math.max(8, window.innerWidth - pet.offsetWidth - 8);
  const place = () => (pet.style.transform = "translateX(" + x + "px)");

  place();
  if (state === "walk") inner.classList.add("walking");

  function hop() {
    if (inner.classList.contains("jump")) return;
    inner.classList.remove("walking");
    inner.classList.add("jump");
    spark();
    inner.addEventListener(
      "animationend",
      function done() {
        inner.classList.remove("jump");
        inner.removeEventListener("animationend", done);
        if (state === "walk") inner.classList.add("walking");
      },
      { once: true }
    );
  }

  function spark() {
    const note = document.createElement("span");
    note.className = "kofun-spark";
    note.textContent = "♪";
    pet.append(note);
    window.setTimeout(() => note.remove(), 900);
  }

  pet.addEventListener("click", hop);
  pet.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      hop();
    }
  });
  window.addEventListener("resize", () => {
    x = Math.min(x, clampMax());
    place();
  });

  // With reduced motion the mascot stays put, but still hops when poked.
  if (reduce) return;

  let stateUntil = 0;
  let last = 0;

  function frame(now) {
    if (!last) last = now;
    const dt = Math.min(48, now - last);
    last = now;
    const max = clampMax();

    if (state === "walk") {
      x += dir * speed * (dt / 16.67);
      if (x <= 8) {
        x = 8;
        dir = 1;
      } else if (x >= max) {
        x = max;
        dir = -1;
      }
      if (Math.random() < 0.004) {
        state = "idle";
        stateUntil = now + 700 + Math.random() * 2200;
        inner.classList.remove("walking");
      }
    } else if (now >= stateUntil) {
      if (Math.random() < 0.5) dir *= -1;
      state = "walk";
      if (!inner.classList.contains("jump")) inner.classList.add("walking");
    }

    place();
    window.requestAnimationFrame(frame);
  }

  window.requestAnimationFrame(frame);
})();
