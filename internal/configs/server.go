package configs

type ServerConfig struct {
	Port      string `env:"SERVER_PORT,required"`
	GRPCPort  string `env:"GRPC_PORT,required"`
	JWTSecret string `env:"JWT_SECRET,required"`
	Audience  string `env:"JWT_AUDIENCE,required"`
}
