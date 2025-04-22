package services

import (
    "context"
    "fmt"
    "log"
    "sync"
    
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

type LangChainService struct {
    llm *openai.LLM
}

func NewLangChainService(apiKey string) (*LangChainService, error) {
    log.Println("Initializing LangChainService with Groq API")
    
    llm, err := openai.New(
        openai.WithToken(apiKey),
        openai.WithBaseURL("https://api.groq.com/openai/v1"),
        openai.WithModel("llama3-70b-8192"),
    )
    if err != nil {
        log.Printf("Error initializing OpenAI LLM: %v", err)
        return nil, err
    }
    
    log.Println("LangChainService initialized successfully")
    return &LangChainService{
        llm: llm,
    }, nil
}

// ProcessConcurrentRequests demonstrates Go's concurrency features
func (s *LangChainService) ProcessConcurrentRequests(ctx context.Context, questions []string) []string {
    log.Printf("Processing %d questions concurrently", len(questions))
    
    var wg sync.WaitGroup
    responses := make([]string, len(questions))
    
    // Create a worker pool for processing requests
    for i, question := range questions {
        wg.Add(1)
        go func(idx int, q string) {
            defer wg.Done()
            log.Printf("Processing question %d: %s", idx, q)
            
            answer, err := s.ProcessSingleRequest(ctx, q)
            if err == nil {
                log.Printf("Question %d answered successfully", idx)
                responses[idx] = answer
            } else {
                log.Printf("Error processing question %d: %v", idx, err)
                responses[idx] = fmt.Sprintf("Error: %v", err)
            }
        }(i, question)
    }
    
    wg.Wait()
    log.Printf("All questions processed, returning %d responses", len(responses))
    return responses
}

func (s *LangChainService) ProcessSingleRequest(ctx context.Context, question string) (string, error) {
    log.Printf("Sending question to LLM: %s", question)
    
    // Prepare the prompt with clear instructions
    prompt := fmt.Sprintf("Answer the following question clearly and concisely: %s", question)
    
    // Call the LLM directly
    completion, err := s.llm.Call(ctx, prompt, llms.WithMaxTokens(2048))
    if err != nil {
        log.Printf("Error calling LLM: %v", err)
        return "", err
    }
    
    log.Printf("Received response from LLM: %s", completion)
    return completion, nil
}

