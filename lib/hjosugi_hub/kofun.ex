defmodule HjosugiHub.Kofun do
  @moduledoc false

  # Single source of truth for the 前方後円墳 (keyhole-tomb) pixel mascot.
  # The same 16x16 grid feeds both the in-page sprite and the favicon, so the
  # artwork is defined once in Elixir and the browser only animates it.

  # Body silhouette as {x, y, width} rows (each one pixel tall): a round rear
  # mound (後円部) on top, a widening trapezoidal front (前方部), then two feet.
  @body [
    {6, 0, 4},
    {5, 1, 6},
    {4, 2, 8},
    {3, 3, 10},
    {3, 4, 10},
    {3, 5, 10},
    {3, 6, 10},
    {4, 7, 8},
    {5, 8, 6},
    {5, 9, 6},
    {4, 10, 8},
    {4, 11, 8},
    {3, 12, 10},
    {2, 13, 12},
    {2, 14, 12},
    {4, 15, 3},
    {9, 15, 3}
  ]

  # Two eyes and a small mouth as {x, y, width, height}.
  @face [
    {5, 4, 1, 2},
    {10, 4, 1, 2},
    {7, 6, 2, 1}
  ]

  @doc "Interactive mascot markup dropped into each page; pet.js only animates it."
  def pet_html do
    """
    <div class="kofun-pet" role="button" tabindex="0" aria-label="poke the kofun — it hops" title="poke me">
    <div class="kofun-sprite">#{sprite_svg()}</div>
    </div>
    """
    |> String.trim()
  end

  @doc "In-page sprite; the body uses currentColor so CSS can theme it."
  def sprite_svg do
    svg(
      ~s(viewBox="0 0 16 16" class="kofun-svg" shape-rendering="crispEdges" aria-hidden="true" focusable="false"),
      group("currentColor", body_rects()) <> group("#0a1c17", face_rects())
    )
  end

  @doc "Standalone favicon: the same pixels on a dark rounded tile."
  def favicon_svg do
    inner =
      ~s|<rect width="64" height="64" rx="12" fill="#08110f"/>| <>
        ~s|<g transform="translate(8 8) scale(3)" shape-rendering="crispEdges">| <>
        group("#62d39c", body_rects()) <> group("#08110f", face_rects()) <> "</g>"

    svg(
      ~s|xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="hjosugi"|,
      inner
    ) <>
      "\n"
  end

  defp svg(attrs, inner), do: "<svg #{attrs}>#{inner}</svg>"

  defp body_rects, do: Enum.map_join(@body, fn {x, y, w} -> rect(x, y, w, 1) end)
  defp face_rects, do: Enum.map_join(@face, fn {x, y, w, h} -> rect(x, y, w, h) end)

  defp group(fill, rects), do: ~s(<g fill="#{fill}">#{rects}</g>)

  defp rect(x, y, w, h), do: ~s(<rect x="#{x}" y="#{y}" width="#{w}" height="#{h}"/>)
end
