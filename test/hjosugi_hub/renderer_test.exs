defmodule HjosugiHub.RendererTest do
  use ExUnit.Case, async: true

  alias HjosugiHub.{Item, Renderer}

  test "export writes radar pages and weighted public data" do
    out_dir =
      Path.join(System.tmp_dir!(), "hjosugi-hub-renderer-#{System.unique_integer([:positive])}")

    site = %{
      handle: "test-hub",
      display_name: "Test Hub",
      headline: "A test site",
      location: "Tokyo, Japan",
      about: "Testing the static export.",
      links: [],
      projects: [],
      skills: []
    }

    feeds = [
      %{id: "hacker-news", name: "Hacker News", kind: "aggregator", enabled: true, tags: []}
    ]

    items = [
      %Item{
        id: "item-1",
        source_id: "hacker-news",
        source_name: "Hacker News",
        source_kind: "aggregator",
        title: "An interesting link",
        url: "https://example.com/item",
        summary: "A summary",
        published_at: ~U[2026-06-20 12:00:00Z],
        collected_at: ~U[2026-06-20 13:00:00Z],
        tags: ["elixir"]
      }
    ]

    try do
      assert :ok = Renderer.export(site, feeds, items, out_dir, "https://example.com/hub/")

      assert File.exists?(Path.join(out_dir, "index.html"))
      assert File.exists?(Path.join(out_dir, "radar/index.html"))
      assert File.exists?(Path.join(out_dir, "popular/index.html"))
      assert File.exists?(Path.join(out_dir, "friends/index.html"))

      radar = File.read!(Path.join(out_dir, "radar/index.html"))
      popular = File.read!(Path.join(out_dir, "popular/index.html"))
      items_json = File.read!(Path.join(out_dir, "data/items.json"))
      sitemap = File.read!(Path.join(out_dir, "sitemap.xml"))

      assert radar =~ ~s(data-category="all")
      assert popular =~ ~s(data-category="github")
      assert items_json =~ ~s("weight":1.3)
      assert sitemap =~ "https://example.com/hub/radar/"
      assert sitemap =~ "https://example.com/hub/popular/"
      assert sitemap =~ "https://example.com/hub/friends/"
    after
      File.rm_rf(out_dir)
    end
  end
end
