// Search index + ranking. Pure data logic, no DOM.
import { collator, norm, tokens, parseTs } from "./radar-util.js";

// Field weights for a matched term and for the whole-query phrase. Keys map to
// the precomputed item._s fields; the entry lists are hoisted out of the loop.
const TERM_WEIGHTS = { t: 7, tags: 5, src: 3, body: 1 };
const PHRASE_WEIGHTS = { t: 6, tags: 4, src: 3, body: 2 };
const TERM_ENTRIES = Object.entries(TERM_WEIGHTS);
const PHRASE_ENTRIES = Object.entries(PHRASE_WEIGHTS);

// Default ordering when nothing is searched: recency biased by source weight,
// plus a popularity nudge from crowd-vote scores (e.g. Hacker News points).
export function baseScore(ts, weight, points) {
  const recency = Math.max(0, ts / 1e13);
  const w = Number(weight) || 1;
  const popularity = points > 0 ? Math.min(1.5, Math.log10(points + 1) / 2) : 0;
  return recency * w + popularity;
}

// Precompute normalized text, a timestamp and a static base score per item so
// ranking on each keystroke is plain comparison, not re-lowercasing long bodies
// or re-parsing dates for every item.
export function prepare(item) {
  const ts = parseTs(item.published_at || item.collected_at);
  item._ts = ts;
  item._date = ts ? new Date(ts).toISOString().slice(0, 10) : "unknown";
  item._base = baseScore(ts, item.weight, item.score);
  item._srcId = norm(item.source_id);
  item._s = {
    t: norm(item.title),
    tags: norm((item.tags || []).join(" ")),
    src: norm(item.source_name),
    body: norm([item.summary, item.content, item.author].filter(Boolean).join(" ")),
  };
  return item;
}

export function rank(values, query) {
  const terms = tokens(query);
  const phrase = norm(query);
  if (terms.length === 0 && !phrase) {
    return [...values].sort((a, b) => b._base - a._base || b._ts - a._ts);
  }
  return [...values]
    .map((item) => ({ item, value: score(item, terms, phrase) }))
    .sort((a, b) => b.value - a.value || b.item._ts - a.item._ts)
    .map((entry) => entry.item);
}

function score(item, terms, phrase) {
  const s = item._s;
  let value = 0;
  for (const term of terms) {
    if (s.t === term) value += 12;
    for (const [field, weight] of TERM_ENTRIES) {
      if (s[field].includes(term)) value += weight;
    }
  }
  if (phrase) {
    for (const [field, weight] of PHRASE_ENTRIES) {
      if (s[field].includes(phrase)) value += weight;
    }
  }
  return value + item._base;
}

export function facets(values, field) {
  const counts = new Map();
  for (const item of values) {
    const entries = field === "tags" ? item.tags || [] : [item[field]].filter(Boolean);
    for (const entry of entries) counts.set(entry, (counts.get(entry) || 0) + 1);
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || collator.compare(a.name, b.name));
}
