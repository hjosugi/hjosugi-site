# Signal Garden

自己紹介ポートフォリオと、自分用の技術情報収集・検索サイトを1つにしたGoアプリです。

## すぐ動かす

```bash
cp .env.example .env
# .env の INBOX_TOKEN と ADMIN_TOKEN を変更
set -a && . ./.env && set +a
go run ./cmd/server
```

- 公開ポートフォリオ: `http://localhost:8080/`
- 非公開の情報収集画面: `http://localhost:8080/inbox`
- ヘルスチェック: `http://localhost:8080/healthz`

ネット接続なしで画面と検索を確認する場合:

```bash
make demo-data
INITIAL_REFRESH=false go run ./cmd/server
```

## 現在できること

- RSS / Atom / YouTube RSSの並行収集
- 重複排除とローカル保存
- クラウド、分散システム、DB、AIなどの自動タグ
- BM25系の全文検索と日本語bigram検索
- Ollamaを使った任意の要約・埋め込み・ハイブリッド検索
- Inboxのトークン保護
- リアルタイム閲覧者数と収集状態
- Docker、CI、テスト、引き継ぎ資料

詳細は [HANDOFF.md](HANDOFF.md) と [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) を参照してください。

## GitHub Pagesで安く公開する

常時起動サーバーなしで公開する場合は、静的版を `public/` に書き出します。
GitHub Actionsが6時間ごとにRSS/Atomを収集し、GitHub Pagesへデプロイします。

```bash
make refresh-feeds
make export-static
```

GitHub PagesではGoサーバーのトークン保護は使えません。`public/data/items.json`
に書き出された情報収集データは公開扱いになります。非公開Inboxが必要なら、Docker版を
VPSやCloudflare Workers系の構成へ出すのが次の選択肢です。
