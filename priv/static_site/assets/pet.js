// The 前方後円墳 (keyhole-tomb) mascot — markup comes from HjosugiHub.Kofun.
// This wires up poke-to-hop and starts the idle wander unless the user prefers
// reduced motion.
import { createWanderer } from "./pet-wander.js";

const pet = document.querySelector(".kofun-pet");
if (pet) {
  const inner = pet.querySelector(".kofun-sprite");
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const wanderer = reduce ? null : createWanderer(pet, inner);
  if (!wanderer) pet.style.transform = "translateX(28px)";

  const spark = () => {
    const note = document.createElement("span");
    note.className = "kofun-spark";
    note.textContent = "♪";
    pet.append(note);
    window.setTimeout(() => note.remove(), 900);
  };

  const hop = () => {
    if (inner.classList.contains("jump")) return;
    wanderer?.pause();
    inner.classList.remove("walking");
    inner.classList.add("jump");
    spark();
    inner.addEventListener(
      "animationend",
      function done() {
        inner.classList.remove("jump");
        inner.removeEventListener("animationend", done);
        wanderer?.resume();
      },
      { once: true }
    );
  };

  pet.addEventListener("click", hop);
  pet.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      hop();
    }
  });
}
