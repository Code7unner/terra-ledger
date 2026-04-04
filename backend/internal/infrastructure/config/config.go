package config

import "time"

type Config struct {
	AppName  string `env:"APP_NAME" envDefault:"terraledger"`
	Port     int    `env:"PORT" envDefault:"3000"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	DatabaseURL string `env:"DATABASE_URL,required"`

	SolanaRPCURL          string `env:"SOLANA_RPC_URL" envDefault:"https://api.devnet.solana.com"`
	TerraTokenProgramID   string `env:"TERRA_TOKEN_PROGRAM_ID,required"`
	LienRegistryProgramID string `env:"LIEN_REGISTRY_PROGRAM_ID,required"`
	RelayKeypairPath      string `env:"RELAY_KEYPAIR_PATH" envDefault:"~/.config/solana/id.json"`
	HeliusAPIKey          string `env:"HELIUS_API_KEY,required"`
	HeliusWebhookSecret   string `env:"HELIUS_WEBHOOK_SECRET,required"`

	CopernicusClientID     string `env:"COPERNICUS_CLIENT_ID"`
	CopernicusClientSecret string `env:"COPERNICUS_CLIENT_SECRET"`

	AnthropicAPIKey string `env:"ANTHROPIC_API_KEY"`
	AnthropicModel  string `env:"ANTHROPIC_MODEL" envDefault:"claude-haiku-4-5-20251001"`

	KeeperInterval time.Duration `env:"KEEPER_INTERVAL" envDefault:"6h"`
}
