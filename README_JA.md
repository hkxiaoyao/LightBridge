<div align="center">

<img src="frontend/public/logo.png" alt="LightBridge" width="120" />

# LightBridge

**セルフホスト型のマルチプロバイダー AI API ゲートウェイ。**

Anthropic・OpenAI・Gemini のアカウントを、OpenAI / Anthropic / Gemini 互換の統一エンドポイントの背後にまとめます。アカウントプール、スマートフェイルオーバー、利用量課金、そして完全な管理コンソールを備えています。

[![Release](https://img.shields.io/github/v/release/WilliamWang1721/LightBridge?style=flat-square)](https://github.com/WilliamWang1721/LightBridge/releases)
[![License: LGPL-3.0](https://img.shields.io/badge/License-LGPL--3.0-blue.svg?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](backend/go.mod)
[![Vue 3](https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vuedotjs)](frontend/package.json)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker)](deploy/DOCKER.md)

[English](README.md) · [简体中文](README_CN.md) · 日本語

</div>

---

## LightBridge とは?

LightBridge はアプリケーションと上流の AI プロバイダーの間に位置します。プロバイダーのアカウント(API キーまたは OAuth)を一度登録するだけで、LightBridge が標準互換のエンドポイントを公開します。正常なアカウントを自動的に選択し、プール間で負荷分散を行い、失敗時にはリトライし、トークン使用量を記録してユーザーに課金します。これらすべてをモダンな Web コンソールから設定できます。

3 大プロバイダーのネイティブプロトコルに対応しているため、既存の SDK やツールをコード変更なしで利用できます:

| プロトコル | エンドポイント | 互換クライアント |
|-----------|---------------|-----------------|
| **Anthropic** | `POST /v1/messages` · `/v1/messages/count_tokens` | Claude SDK、Claude Code、Anthropic クライアント |
| **OpenAI** | `POST /v1/chat/completions` · `/v1/responses` | OpenAI SDK、Codex、各種 OpenAI 互換クライアント |
| **Gemini** | `POST /v1beta/models/{model}:generateContent` | Google GenAI SDK、Gemini CLI |

## 主な機能

**🔌 マルチプロバイダーゲートウェイ**
- 単一ホストから Anthropic / OpenAI / Gemini 互換 API を提供
- カスタムプロバイダー(任意の OpenAI 互換上流)に対応
- モデルごとのマッピングとホワイトリスト

**⚖️ アカウントプールと高可用性**
- プロバイダーごとに複数アカウントを設定し、優先度・重み・負荷係数をサポート
- 自動負荷分散とヘルスベースのアカウント選択
- フェイルオーバーループ:リクエスト失敗時に他の正常なアカウントへ自動的に切り替えてリトライ
- チャネル監視と 30 日間の GitHub 風可用性グリッド

**🔐 柔軟な認証**
- API キー(API キー認証付き)、Gemini は OAuth(Code Assist / AI Studio / API Key)に対応
- ユーザーログインはメール、LinuxDO、Google/GitHub、WeChat、DingTalk、汎用 OIDC に対応

**💳 課金とマルチテナント**
- ユーザーごとの API キー、クォータ、同時実行数の制限
- トークンベースの使用量追跡、価格と課金倍率を設定可能
- Stripe / Airwallex 決済連携、招待リベートに対応

**🛡️ プライバシーとセキュリティ**
- 組み込みのプライバシーフィルター(IPv6、JWT、PEM 秘密鍵、AWS/GitHub/Slack トークン、クレジットカード番号など)、ユーザーとチャネル単位で適用
- コンテンツモデレーションフック、上流リクエストへの TLS フィンガープリント模倣

**📊 管理コンソール**
- カスタマイズ可能・ドラッグ&ドロップ対応のダッシュボードカード:可用性、同時実行、スループット、レイテンシ、エラー傾向、トークン使用量、モデル分布など
- ユーザーとアカウントの一括管理、お知らせ、アラート、システムログ
- 組み込み機能をオンデマンドで有効/無効にできるモジュールマーケットプレイス

<!-- PLACEHOLDER_JA -->
