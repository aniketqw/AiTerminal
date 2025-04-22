package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "my-groq-project/services"
)

type AIHandler struct {
    langchainService *services.LangChainService
}

func NewAIHandler(ls *services.LangChainService) *AIHandler {
    return &AIHandler{langchainService: ls}
}

type QuestionRequest struct {
    Questions []string `json:"questions"`
}

func (h *AIHandler) HandleBatchQuestions(c *gin.Context) {
    var req QuestionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    responses := h.langchainService.ProcessConcurrentRequests(c.Request.Context(), req.Questions)
    c.JSON(http.StatusOK, gin.H{"responses": responses})
}

