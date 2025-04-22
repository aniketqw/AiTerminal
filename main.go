package main
// go insall github.com/gdamore/tcell/v2@latest

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "my-groq-project/config"
    "my-groq-project/handlers"
    "my-groq-project/services"
)

func startServer() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Printf("Warning: Error loading .env file: %v", err)
    }
    
    // Load configuration
    cfg := config.LoadConfig()
    
    // Initialize LangChain service with Groq
    langchainService, err := services.NewLangChainService(cfg.GroqKey)
    if err != nil {
        log.Fatalf("Failed to initialize LangChain service: %v", err)
    }
    
    // Initialize handler
    aiHandler := handlers.NewAIHandler(langchainService)
    
    // Setup Gin router
    router := gin.Default()
    
    // Define routes
    router.POST("/api/questions", aiHandler.HandleBatchQuestions)
    
    // Start server
    router.Run(":" + cfg.Port)
}

