defmodule HjosugiHub.Renderer do
  @moduledoc false

  alias HjosugiHub.{Config, Kofun, Store}

  @template_dir Path.expand("../../priv/static_site/templates", __DIR__)
  @asset_dir Path.expand("../../priv/static_site/assets", __DIR__)

  def export(site, feeds, items, out_dir, base_url \\ "") do
    weights = Map.new(feeds, fn feed -> {feed.id, Config.feed_weight(feed)} end)

    public_items =
      items
      |> Store.public_items()
      |> Enum.map(fn item -> Map.put(item, :weight, Map.get(weights, item.source_id, 1.0)) end)

    now = DateTime.utc_now()

    assigns = %{
      site: site,
      feeds: feeds,
      enabled_feeds: Config.enabled_feeds(feeds),
      featured: Enum.filter(Map.get(site, :projects, []), &Map.get(&1, :featured, false)),
      others: Enum.reject(Map.get(site, :projects, []), &Map.get(&1, :featured, false)),
      avatar_url: Config.avatar_url(site),
      kofun: Kofun.pet_html(),
      items: public_items,
      generated_text: Calendar.strftime(now, "%Y-%m-%d %H:%M UTC"),
      year: now.year,
      base_url: String.trim_trailing(base_url || "", "/")
    }

    write_rendered(out_dir, "index.html", "index.html.eex", assigns)
    write_radar_pages(out_dir, assigns)
    Store.write_json(Path.join(out_dir, "data/items.json"), public_items)
    Store.write_json(Path.join(out_dir, "data/site.json"), site)
    Store.write_json(Path.join(out_dir, "data/feeds.json"), public_feeds(feeds))
    copy_assets(out_dir)
    File.write!(Path.join(out_dir, "static/favicon.svg"), Kofun.favicon_svg())
    File.write!(Path.join(out_dir, ".nojekyll"), "")
    File.write!(Path.join(out_dir, "robots.txt"), robots(assigns.base_url))

    if assigns.base_url != "" do
      File.write!(Path.join(out_dir, "sitemap.xml"), sitemap(assigns.base_url))
    end

    :ok
  end

  # The radar is split into category-scoped pages with shared tabs. The full
  # view lives at /radar/; /radar/github/ and /radar/news/ filter client-side
  # (by link host and aggregator source kind). `root` is the relative path back
  # to the site root so assets, data, and tab links resolve from each depth.
  @radar_pages [
    {"radar", "all", "../"},
    {"radar/github", "github", "../../"},
    {"radar/news", "news", "../../"}
  ]

  defp write_radar_pages(out_dir, assigns) do
    Enum.each(@radar_pages, fn {path, category, root} ->
      scoped = Map.merge(assigns, %{category: category, root: root})
      write_rendered(Path.join(out_dir, path), "index.html", "radar.html.eex", scoped)
    end)
  end

  defp write_rendered(dir, file, template, assigns) do
    File.mkdir_p!(dir)
    html = EEx.eval_file(Path.join(@template_dir, template), assigns: assigns)
    File.write!(Path.join(dir, file), html)
  end

  defp copy_assets(out_dir) do
    target = Path.join(out_dir, "static")
    File.mkdir_p!(target)

    @asset_dir
    |> Path.join("*")
    |> Path.wildcard()
    |> Enum.each(fn path -> File.cp!(path, Path.join(target, Path.basename(path))) end)
  end

  defp public_feeds(feeds) do
    feeds
    |> Enum.map(&Map.take(&1, [:id, :name, :kind, :enabled, :tags]))
    |> Enum.sort_by(& &1.name)
  end

  defp robots(""), do: "User-agent: *\nAllow: /\n"
  defp robots(base_url), do: "User-agent: *\nAllow: /\nSitemap: #{base_url}/sitemap.xml\n"

  defp sitemap(base_url) do
    """
    <?xml version="1.0" encoding="UTF-8"?>
    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
      <url><loc>#{base_url}/</loc></url>
      <url><loc>#{base_url}/radar/</loc></url>
    </urlset>
    """
    |> String.trim_leading()
  end
end
