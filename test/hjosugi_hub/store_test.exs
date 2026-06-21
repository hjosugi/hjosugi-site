defmodule HjosugiHub.StoreTest do
  use ExUnit.Case, async: true

  alias HjosugiHub.{Item, Store}

  test "normalizes cached items from the previous app name" do
    path =
      Path.join(System.tmp_dir!(), "hjosugi-hub-store-#{System.unique_integer([:positive])}.term")

    previous_struct = String.to_atom("Elixir.Legacy.Item")

    legacy_items = [
      %{
        __struct__: previous_struct,
        id: "legacy-1",
        source_id: "source",
        source_name: "Source",
        source_kind: "rss",
        title: "Cached item",
        url: "https://example.com/item",
        author: "",
        summary: "summary",
        content: "content",
        published_at: ~U[2026-06-20 12:00:00Z],
        collected_at: ~U[2026-06-21 00:00:00Z],
        tags: ["cache"]
      }
    ]

    File.write!(path, :erlang.term_to_binary(legacy_items))

    assert [%Item{} = item] = Store.read_items(path)
    assert item.id == "legacy-1"
    assert item.tags == ["cache"]

    File.rm(path)
  end

  test "reads cached items serialized before :score existed without crashing" do
    path =
      Path.join(System.tmp_dir!(), "hjosugi-hub-store-#{System.unique_integer([:positive])}.term")

    # A HjosugiHub.Item struct as serialized before the :score field was added.
    legacy_item =
      %{
        __struct__: Item,
        id: "old-1",
        source_id: "hacker-news",
        source_name: "Hacker News",
        source_kind: "aggregator",
        title: "Cached HN item",
        url: "https://example.com/item",
        author: "someone",
        summary: "summary",
        content: "content",
        published_at: ~U[2026-06-20 12:00:00Z],
        collected_at: ~U[2026-06-21 00:00:00Z],
        tags: ["aggregator"]
      }

    File.write!(path, :erlang.term_to_binary([legacy_item]))

    items = Store.read_items(path)
    assert [%Item{score: nil}] = items
    assert [%{score: nil}] = Store.public_items(items)

    File.rm(path)
  end

  test "public_items/1 tolerates a struct missing :score without crashing" do
    # Reproduces the hub.collect crash: public_items is called directly on
    # merged items (not via read_items), so a struct deserialized from the
    # cache before :score existed reaches it without normalization.
    stale =
      %{
        __struct__: Item,
        id: "stale-1",
        source_id: "hacker-news",
        source_name: "Hacker News Front Page",
        source_kind: "aggregator",
        title: "stale item",
        url: "https://github.com/owner/repo",
        author: "someone",
        summary: "summary",
        content: "content",
        published_at: ~U[2026-06-21 00:00:00Z],
        collected_at: ~U[2026-06-21 01:31:00Z],
        tags: ["aggregator"]
      }

    refute Map.has_key?(stale, :score)
    assert [%{id: "stale-1", score: nil}] = Store.public_items([stale])
  end
end
