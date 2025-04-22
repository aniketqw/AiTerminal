package config

import "os"

type Config struct {
    GroqKey string
    Port    string
}

func LoadConfig() *Config {
    // Get Groq API key, with GROQ_API_KEY taking precedence over OPENAI_API_KEY
    groqKey := os.Getenv("GROQ_API_KEY")
    if groqKey == "" {
        groqKey = os.Getenv("OPENAI_API_KEY")
    }
    
    // If the GROQ_API_KEY is set, also set it as OPENAI_API_KEY for the OpenAI client
    if groqKey != "" && os.Getenv("OPENAI_API_KEY") == "" {
        os.Setenv("OPENAI_API_KEY", groqKey)
    }
    
    return &Config{
        GroqKey: groqKey,
        Port:    os.Getenv("PORT"),
    }
}

