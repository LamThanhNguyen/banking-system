package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/spf13/viper"
)

type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT" json:"ENVIRONMENT"`
	AllowedOrigins       []string      `mapstructure:"ALLOWED_ORIGINS" json:"ALLOWED_ORIGINS"`
	DBSource             string        `mapstructure:"DB_SOURCE" json:"DB_SOURCE"`
	MigrationURL         string        `mapstructure:"MIGRATION_URL" json:"MIGRATION_URL"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS" json:"REDIS_ADDRESS"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS" json:"HTTP_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY" json:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION" json:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION" json:"REFRESH_TOKEN_DURATION"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME" json:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS" json:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD" json:"EMAIL_SENDER_PASSWORD"`
	FrontendDomain       string        `mapstructure:"FRONTEND_DOMAIN" json:"FRONTEND_DOMAIN"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(ctx context.Context, path string) (Config, error) {
	environment := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if environment == "" {
		environment = "develop"
	}
	fmt.Printf("Loading config for environment: %s\n", environment)

	var config Config

	switch environment {
	case "develop":
		viper.AddConfigPath(path)
		viper.SetConfigFile(".env") // Specify exact file name
		viper.SetConfigType("env")
		_ = viper.ReadInConfig()
		viper.AutomaticEnv()
		err := viper.Unmarshal(&config)
		if err != nil {
			return config, fmt.Errorf("viper unmarshal error: %w", err)
		}
		return config, nil

	case "staging", "production":
		// secretName := fmt.Sprintf("%s/banking-system", environment)
		secretName := "banking-system"
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion("ap-southeast-1"))
		if err != nil {
			return config, fmt.Errorf("unable to load AWS config: %w", err)
		}
		svc := secretsmanager.NewFromConfig(awsCfg)
		secretValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
		if err != nil {
			return config, fmt.Errorf("failed to get secret: %w", err)
		}
		err = json.Unmarshal([]byte(*secretValue.SecretString), &config)
		if err != nil {
			return config, fmt.Errorf("unmarshal secret: %w", err)
		}
		return config, nil

	default:
		return config, errors.New("invalid ENVIRONMENT: must be one of develop/staging/production")
	}
}
