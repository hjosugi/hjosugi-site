// The mascot's idle wander loop: it strolls along the bottom and pauses now and
// then. Returns pause()/resume() so the interaction layer can stop it mid-hop.
export function createWanderer(pet, inner) {
  const speed = 0.42; // px per ~16ms frame
  let x = 28;
  let dir = 1;
  let state = "walk";
  let stateUntil = 0;
  let last = 0;
  let paused = false;

  const clampMax = () => Math.max(8, window.innerWidth - pet.offsetWidth - 8);
  const place = () => (pet.style.transform = "translateX(" + x + "px)");

  place();
  inner.classList.add("walking");
  window.addEventListener("resize", () => {
    x = Math.min(x, clampMax());
    place();
  });

  function frame(now) {
    if (!last) last = now;
    const dt = Math.min(48, now - last);
    last = now;

    if (!paused) {
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
        inner.classList.add("walking");
      }
      place();
    }

    window.requestAnimationFrame(frame);
  }

  window.requestAnimationFrame(frame);

  return {
    pause() {
      paused = true;
      inner.classList.remove("walking");
    },
    resume() {
      paused = false;
      if (state === "walk") inner.classList.add("walking");
    },
  };
}
