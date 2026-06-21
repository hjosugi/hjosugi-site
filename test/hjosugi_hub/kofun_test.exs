defmodule HjosugiHub.KofunTest do
  use ExUnit.Case, async: true

  alias HjosugiHub.Kofun

  test "pet markup wraps the themeable sprite" do
    html = Kofun.pet_html()
    assert html =~ ~s(class="kofun-pet")
    assert html =~ ~s(role="button")
    assert html =~ ~s(<svg)
    assert html =~ "currentColor"
  end

  test "favicon is a standalone svg using the same pixel rows" do
    favicon = Kofun.favicon_svg()
    assert favicon =~ ~s(xmlns="http://www.w3.org/2000/svg")
    # Both come from one grid; the favicon only adds a background tile rect.
    rects = fn svg -> svg |> String.split("<rect") |> length() end
    assert rects.(favicon) == rects.(Kofun.sprite_svg()) + 1
  end
end
