package service

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"search-service/internal/entity"
)

// ============================================================================
// PART 1: AI SUGGESTION SERVICE (US-010)
// ============================================================================

type AISuggestionService interface {
	GenerateAISuggestions(ctx context.Context) error
}

type aiSuggestionService struct {
	repo     SearchRepository
	analyzer KeywordAnalyzer
}

func NewAISuggestionService(repo SearchRepository, analyzer KeywordAnalyzer) AISuggestionService {
	return &aiSuggestionService{
		repo:     repo,
		analyzer: analyzer,
	}
}

func (s *aiSuggestionService) GenerateAISuggestions(ctx context.Context) error {
	// Fetch active tenants from logs
	tenants, err := s.repo.GetActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active tenants: %w", err)
	}

	for _, tenantID := range tenants {
		// Fetch top 50 zero-result queries
		zeroQueries, err := s.repo.GetZeroResultQueries(ctx, tenantID, 100)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch zero-result queries for tenant %s: %v", tenantID, err)
			continue
		}

		// Fetch top 50 low-CTR queries
		lowCTRQueries, err := s.repo.GetLowCTRQueries(ctx, tenantID, 50)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch low-CTR queries for tenant %s: %v", tenantID, err)
			continue
		}

		// Aggregate queries to process
		queryMap := make(map[string]bool)
		var keywords []string
		for _, q := range zeroQueries {
			if len(keywords) < 100 && !queryMap[q.Query] {
				queryMap[q.Query] = true
				keywords = append(keywords, q.Query)
			}
		}
		for _, q := range lowCTRQueries {
			if len(keywords) < 100 && !queryMap[q.Query] {
				queryMap[q.Query] = true
				keywords = append(keywords, q.Query)
			}
		}

		if len(keywords) == 0 {
			continue
		}

		// Fetch tenant catalog summary context for the LLM
		tenantContext, err := s.repo.GetTenantContextSummary(ctx, tenantID)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to fetch context summary for tenant %s: %v", tenantID, err)
			continue
		}

		// Call OpenAI to analyze the keywords
		suggestions, err := s.analyzer.AnalyzeKeywords(ctx, keywords, tenantContext)
		if err != nil {
			log.Printf("[AISuggestionWorker] Failed to analyze keywords for tenant %s: %v", tenantID, err)
			continue
		}

		// Save suggestions into DB
		for _, sugg := range suggestions {
			sugg.TenantID = tenantID
			sugg.Status = "pending"
			if err := s.repo.SaveAISuggestion(ctx, &sugg); err != nil {
				log.Printf("[AISuggestionWorker] Failed to save suggestion for tenant %s (from '%s' to '%s'): %v",
					tenantID, sugg.SourceValue, sugg.SuggestedValue, err)
			}
		}
	}

	return nil
}

// ============================================================================
// PART 2: ADMIN AI ASSISTANT SERVICE (US-013)
// ============================================================================

type AssistantService interface {
	GetConversations(ctx context.Context, tenantID string) ([]entity.AssistantConversation, error)
	CreateConversation(ctx context.Context, tenantID, title string) (*entity.AssistantConversation, error)
	GetConversationMessages(ctx context.Context, convID string) ([]entity.ChatMessage, error)
	DeleteConversation(ctx context.Context, id string) error
	ChatWithAssistant(ctx context.Context, tenantID, conversationID, message string) (string, []entity.ProposedAction, string, error)
	UpdateActionState(ctx context.Context, msgID string, actionIndex int, state string) error
}

type assistantService struct {
	repo   SearchRepository
	apiKey string
	model  string
	client *http.Client
}

func NewAssistantService(repo SearchRepository, apiKey, model string) AssistantService {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &assistantService{
		repo:   repo,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 35 * time.Second,
		},
	}
}

type openAIChatMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = cryptoRand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *assistantService) GetConversations(ctx context.Context, tenantID string) ([]entity.AssistantConversation, error) {
	return s.repo.GetConversations(ctx, tenantID)
}

func (s *assistantService) CreateConversation(ctx context.Context, tenantID, title string) (*entity.AssistantConversation, error) {
	if title == "" {
		title = "Hội thoại mới"
	}
	conv := &entity.AssistantConversation{
		ID:       generateUUID(),
		TenantID: tenantID,
		Title:    title,
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *assistantService) GetConversationMessages(ctx context.Context, convID string) ([]entity.ChatMessage, error) {
	dbMsgs, err := s.repo.GetConversationMessages(ctx, convID)
	if err != nil {
		return nil, err
	}

	var chatMsgs []entity.ChatMessage
	for _, m := range dbMsgs {
		var proposed []entity.ProposedAction
		if m.ProposedActions != "" {
			_ = json.Unmarshal([]byte(m.ProposedActions), &proposed)
		}
		var states map[string]string
		if m.ActionStates != "" {
			_ = json.Unmarshal([]byte(m.ActionStates), &states)
		}

		chatMsgs = append(chatMsgs, entity.ChatMessage{
			ID:              m.ID,
			Role:            m.Role,
			Content:         m.Content,
			ProposedActions: proposed,
			ActionStates:    states,
			CreatedAt:       m.CreatedAt,
		})
	}
	return chatMsgs, nil
}

func (s *assistantService) DeleteConversation(ctx context.Context, id string) error {
	return s.repo.DeleteConversation(ctx, id)
}

func (s *assistantService) UpdateActionState(ctx context.Context, msgID string, actionIndex int, state string) error {
	msg, err := s.repo.GetAssistantMessageByID(ctx, msgID)
	if err != nil {
		return err
	}

	states := make(map[string]string)
	if msg.ActionStates != "" {
		_ = json.Unmarshal([]byte(msg.ActionStates), &states)
	}

	idxStr := fmt.Sprintf("%d", actionIndex)
	states[idxStr] = state

	statesBytes, _ := json.Marshal(states)
	return s.repo.UpdateMessageActionStates(ctx, msgID, string(statesBytes))
}

func (s *assistantService) ChatWithAssistant(ctx context.Context, tenantID, conversationID, message string) (string, []entity.ProposedAction, string, error) {
	// 1. Fetch message history for the active conversation
	dbMsgs, err := s.repo.GetConversationMessages(ctx, conversationID)
	if err != nil {
		return "", nil, "", err
	}

	// 2. Save the incoming user message to DB
	userMsgID := generateUUID()
	userMsg := &entity.AssistantMessage{
		ID:             userMsgID,
		ConversationID: conversationID,
		Role:           "user",
		Content:        message,
	}
	if err := s.repo.SaveAssistantMessage(ctx, userMsg); err != nil {
		return "", nil, "", err
	}

	// Translate history to OpenAI message format
	var history []entity.ChatMessage
	for _, m := range dbMsgs {
		history = append(history, entity.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	var finalReply string
	var proposedActions []entity.ProposedAction

	if s.apiKey == "" || strings.HasPrefix(s.apiKey, "YOUR_") {
		// Mock assistant response for local testing when API key is missing
		var mockActions []entity.ProposedAction
		finalReply, mockActions = s.mockLocalAssistantResponse(ctx, tenantID, message)
		proposedActions = mockActions
	} else {
		// Call OpenAI Chat API using message history
		var err error
		finalReply, proposedActions, err = s.callOpenAIWithHistory(ctx, tenantID, message, history)
		if err != nil {
			return "", nil, "", err
		}
	}

	// 3. Save assistant response message to DB
	assistantMsgID := generateUUID()
	proposedJSON := "[]"
	if len(proposedActions) > 0 {
		pBytes, _ := json.Marshal(proposedActions)
		proposedJSON = string(pBytes)
	}

	assistantMsg := &entity.AssistantMessage{
		ID:              assistantMsgID,
		ConversationID:  conversationID,
		Role:            "assistant",
		Content:         finalReply,
		ProposedActions: proposedJSON,
		ActionStates:    "{}",
	}
	if err := s.repo.SaveAssistantMessage(ctx, assistantMsg); err != nil {
		return "", nil, "", err
	}

	// 4. Update conversation title on the first message
	if len(dbMsgs) == 0 {
		title := message
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		_ = s.repo.UpdateConversationTitle(ctx, conversationID, title)
	}

	return finalReply, proposedActions, assistantMsgID, nil
}

func (s *assistantService) callOpenAIWithHistory(ctx context.Context, tenantID, message string, history []entity.ChatMessage) (string, []entity.ProposedAction, error) {
	// Build System Prompt with Tenant Grounding Context
	tenantSummary, err := s.repo.GetTenantContextSummary(ctx, tenantID)
	if err != nil {
		log.Printf("[AssistantService] Failed to load tenant context summary: %v", err)
		tenantSummary = "Tenant ID: " + tenantID
	}

	systemPrompt := fmt.Sprintf(`You are the Swift Search Engine Admin Assistant, a helpful AI Agent for the e-commerce store administrator.
Your goal is to help the administrator manage the e-commerce search engine dictionaries and search products.

Tenant context for the current store:
%s

Core instructions:
1. You can search products in the store's inventory and view synonyms/spellcheck rules by calling the provided tools.
2. For any write actions (creating or deleting synonym/spellcheck rules), you MUST call the respective proposal tools (e.g. create_synonym, delete_synonym, create_spellcheck, delete_spellcheck). Explain to the user that these are proposed actions and they must click the "Accept" button on the screen to apply them.
3. Be concise, polite, helpful, and speak in Vietnamese as the primary language (matching the administrator's language).
4. Strictly operate only within the scope of the active tenant. Never leak or guess information outside the tenant context.
`, tenantSummary)

	// Prepare messages for OpenAI
	var messages []openAIChatMessage
	messages = append(messages, openAIChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	for _, h := range history {
		role := h.Role
		if role != "user" && role != "assistant" && role != "system" {
			role = "user"
		}
		messages = append(messages, openAIChatMessage{
			Role:    role,
			Content: h.Content,
		})
	}

	messages = append(messages, openAIChatMessage{
		Role:    "user",
		Content: message,
	})

	tools := assistantTools
	var finalReply string
	var proposedActions []entity.ProposedAction

	// OpenAI Request Loop
	for turn := 0; turn < 3; turn++ {
		reqBody, err := json.Marshal(map[string]interface{}{
			"model":       s.model,
			"messages":    messages,
			"tools":       tools,
			"temperature": 0.3,
		})
		if err != nil {
			return "", nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
		if err != nil {
			return "", nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

		resp, err := s.client.Do(req)
		if err != nil {
			return "", nil, fmt.Errorf("failed to call OpenAI Chat API: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", nil, fmt.Errorf("failed to read OpenAI response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("[AssistantService] OpenAI Error (%d): %s", resp.StatusCode, string(respBody))
			return "", nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBody))
		}

		var chatResp struct {
			Choices []struct {
				Message struct {
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []openAIToolCall `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			return "", nil, err
		}

		if len(chatResp.Choices) == 0 {
			return "", nil, fmt.Errorf("empty choice from OpenAI response")
		}

		choiceMessage := chatResp.Choices[0].Message
		messages = append(messages, openAIChatMessage{
			Role:      choiceMessage.Role,
			Content:   choiceMessage.Content,
			ToolCalls: choiceMessage.ToolCalls,
		})

		if len(choiceMessage.ToolCalls) == 0 {
			finalReply = choiceMessage.Content
			break
		}

		// Process tool calls
		for _, tc := range choiceMessage.ToolCalls {
			toolResult, action, err := s.executeTool(ctx, tenantID, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				log.Printf("[AssistantService] Error executing tool %s: %v", tc.Function.Name, err)
				toolResult = fmt.Sprintf(`{"status": "error", "error": "%v"}`, err)
			}

			if action != nil {
				proposedActions = append(proposedActions, *action)
			}

			messages = append(messages, openAIChatMessage{
				Role:       "tool",
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
				Content:    toolResult,
			})
		}
	}

	return finalReply, proposedActions, nil
}

func (s *assistantService) executeTool(ctx context.Context, tenantID string, name string, arguments string) (string, *entity.ProposedAction, error) {
	log.Printf("[AssistantService] Executing Tool: %s with args: %s", name, arguments)

	switch name {
	case "search_products":
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", nil, err
		}
		if args.Limit <= 0 {
			args.Limit = 5
		}

		products, err := s.repo.GetAllProductsByTenantID(ctx, tenantID)
		if err != nil {
			return "", nil, err
		}

		queryLower := strings.ToLower(args.Query)
		var matches []map[string]interface{}
		for _, p := range products {
			if strings.Contains(strings.ToLower(p.Name), queryLower) || strings.Contains(strings.ToLower(p.Brand), queryLower) {
				matches = append(matches, map[string]interface{}{
					"id":        p.ID,
					"name":      p.Name,
					"brand":     p.Brand,
					"price":     p.Price,
					"inventory": p.Inventory,
					"status":    p.Status,
					"featured":  p.Featured,
				})
			}
			if len(matches) >= args.Limit {
				break
			}
		}

		resBytes, _ := json.Marshal(matches)
		return string(resBytes), nil, nil

	case "get_active_dictionaries":
		synonyms, err := s.repo.GetSearchSynonyms(ctx, tenantID)
		if err != nil {
			return "", nil, err
		}
		spellchecks, err := s.repo.GetSpellcheckRules(ctx, tenantID)
		if err != nil {
			return "", nil, err
		}

		result := map[string]interface{}{
			"synonyms":    synonyms,
			"spellchecks": spellchecks,
		}
		resBytes, _ := json.Marshal(result)
		return string(resBytes), nil, nil

	case "create_synonym":
		var args struct {
			Keyword         string `json:"keyword"`
			Synonym         string `json:"synonym"`
			IsBidirectional bool   `json:"is_bidirectional"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", nil, err
		}

		desc := fmt.Sprintf("Thêm từ đồng nghĩa '%s' -> '%s' (một chiều)", args.Keyword, args.Synonym)
		if args.IsBidirectional {
			desc = fmt.Sprintf("Thêm từ đồng nghĩa '%s' <-> '%s' (hai chiều)", args.Keyword, args.Synonym)
		}

		action := &entity.ProposedAction{
			ActionType:  "create_synonym",
			Description: desc,
			Params: map[string]interface{}{
				"keyword":          args.Keyword,
				"synonym":          args.Synonym,
				"is_bidirectional": args.IsBidirectional,
			},
		}

		res := map[string]string{
			"status":  "proposed",
			"message": "Synonym creation proposed to the admin for review.",
		}
		resBytes, _ := json.Marshal(res)
		return string(resBytes), action, nil

	case "delete_synonym":
		var args struct {
			Keyword string `json:"keyword"`
			Synonym string `json:"synonym"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", nil, err
		}

		synRules, err := s.repo.GetSearchSynonyms(ctx, tenantID)
		if err != nil {
			return "", nil, err
		}

		var ids []string
		kwLower := strings.ToLower(strings.TrimSpace(args.Keyword))
		synLower := strings.ToLower(strings.TrimSpace(args.Synonym))

		for _, r := range synRules {
			rKw := strings.ToLower(r.Keyword)
			rSyn := strings.ToLower(r.Synonym)
			if (rKw == kwLower && rSyn == synLower) || (rKw == synLower && rSyn == kwLower) {
				ids = append(ids, r.ID)
			}
		}

		if len(ids) == 0 {
			res := map[string]interface{}{
				"status":  "not_found",
				"message": fmt.Sprintf("No active synonym rules found matching '%s' and '%s'.", args.Keyword, args.Synonym),
			}
			resBytes, _ := json.Marshal(res)
			return string(resBytes), nil, nil
		}

		action := &entity.ProposedAction{
			ActionType:  "delete_synonym",
			Description: fmt.Sprintf("Xóa các quy tắc từ đồng nghĩa liên kết giữa '%s' và '%s'", args.Keyword, args.Synonym),
			Params: map[string]interface{}{
				"ids":     ids,
				"keyword": args.Keyword,
				"synonym": args.Synonym,
			},
		}

		res := map[string]interface{}{
			"status":  "proposed",
			"message": fmt.Sprintf("Synonym deletion proposed for %d rules matching '%s' and '%s'.", len(ids), args.Keyword, args.Synonym),
		}
		resBytes, _ := json.Marshal(res)
		return string(resBytes), action, nil

	case "create_spellcheck":
		var args struct {
			TypoWord    string `json:"typo_word"`
			CorrectWord string `json:"correct_word"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", nil, err
		}

		action := &entity.ProposedAction{
			ActionType:  "create_spellcheck",
			Description: fmt.Sprintf("Sửa lỗi chính tả gõ sai '%s' thành '%s'", args.TypoWord, args.CorrectWord),
			Params: map[string]interface{}{
				"typo_word":    args.TypoWord,
				"correct_word": args.CorrectWord,
			},
		}

		res := map[string]string{
			"status":  "proposed",
			"message": "Spellcheck creation proposed to the admin for review.",
		}
		resBytes, _ := json.Marshal(res)
		return string(resBytes), action, nil

	case "delete_spellcheck":
		var args struct {
			TypoWord    string `json:"typo_word"`
			CorrectWord string `json:"correct_word"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", nil, err
		}

		rules, err := s.repo.GetSpellcheckRules(ctx, tenantID)
		if err != nil {
			return "", nil, err
		}

		var ids []string
		typoLower := strings.ToLower(strings.TrimSpace(args.TypoWord))
		correctLower := strings.ToLower(strings.TrimSpace(args.CorrectWord))

		for _, r := range rules {
			rTypo := strings.ToLower(r.TypoWord)
			rCorr := strings.ToLower(r.CorrectWord)
			if rTypo == typoLower && (correctLower == "" || rCorr == correctLower) {
				ids = append(ids, r.ID)
			}
		}

		if len(ids) == 0 {
			res := map[string]interface{}{
				"status":  "not_found",
				"message": fmt.Sprintf("No active spellcheck rules found for typo '%s'.", args.TypoWord),
			}
			resBytes, _ := json.Marshal(res)
			return string(resBytes), nil, nil
		}

		action := &entity.ProposedAction{
			ActionType:  "delete_spellcheck",
			Description: fmt.Sprintf("Xóa quy tắc sửa chính tả cho từ gõ sai '%s'", args.TypoWord),
			Params: map[string]interface{}{
				"ids":       ids,
				"typo_word": args.TypoWord,
			},
		}

		res := map[string]interface{}{
			"status":  "proposed",
			"message": fmt.Sprintf("Spellcheck deletion proposed for %d rules matching typo '%s'.", len(ids), args.TypoWord),
		}
		resBytes, _ := json.Marshal(res)
		return string(resBytes), action, nil

	default:
		return "", nil, fmt.Errorf("unknown tool function: %s", name)
	}
}

func (s *assistantService) mockLocalAssistantResponse(ctx context.Context, tenantID string, message string) (string, []entity.ProposedAction) {
	msg := strings.ToLower(message)
	var reply string
	var actions []entity.ProposedAction

	if strings.Contains(msg, "đồng nghĩa") && (strings.Contains(msg, "thêm") || strings.Contains(msg, "tạo")) {
		// Mock Synonym creation proposal
		reply = "Tôi đã chuẩn bị đề xuất tạo từ đồng nghĩa mới theo yêu cầu của bạn. Bạn vui lòng xác nhận và nhấn Chấp nhận trên thẻ hành động bên dưới để áp dụng vào hệ thống."
		actions = append(actions, entity.ProposedAction{
			ActionType:  "create_synonym",
			Description: "Thêm từ đồng nghĩa 'bàn phím cơ' <-> 'phím cơ' (hai chiều) [MOCK LOCAL]",
			Params: map[string]interface{}{
				"keyword":          "bàn phím cơ",
				"synonym":          "phím cơ",
				"is_bidirectional": true,
			},
		})
	} else if strings.Contains(msg, "chính tả") && (strings.Contains(msg, "thêm") || strings.Contains(msg, "tạo") || strings.Contains(msg, "sửa")) {
		// Mock Spellcheck creation proposal
		reply = "Tôi đã soạn sẵn đề xuất sửa chính tả. Bạn vui lòng kiểm tra xem từ gõ sai và từ gõ đúng đã chính xác chưa, rồi bấm Chấp nhận để kích hoạt quy tắc nhé."
		actions = append(actions, entity.ProposedAction{
			ActionType:  "create_spellcheck",
			Description: "Sửa lỗi chính tả gõ sai 'ako' thành 'akko' [MOCK LOCAL]",
			Params: map[string]interface{}{
				"typo_word":    "ako",
				"correct_word": "akko",
			},
		})
	} else if strings.Contains(msg, "xóa") && strings.Contains(msg, "đồng nghĩa") {
		// Mock Synonym delete proposal
		reply = "Tôi đã tìm thấy quy tắc từ đồng nghĩa liên kết và chuẩn bị đề xuất xóa. Hãy nhấn Chấp nhận để thực thi."
		actions = append(actions, entity.ProposedAction{
			ActionType:  "delete_synonym",
			Description: "Xóa các quy tắc từ đồng nghĩa liên kết giữa 'máy tính' và 'computer' [MOCK LOCAL]",
			Params: map[string]interface{}{
				"ids":     []string{"mock-synonym-id-123"},
				"keyword": "máy tính",
				"synonym": "computer",
			},
		})
	} else if strings.Contains(msg, "xóa") && (strings.Contains(msg, "chính tả") || strings.Contains(msg, "typo")) {
		// Mock Spellcheck delete proposal
		reply = "Tôi đã tìm thấy quy tắc sửa chính tả khớp và lập đề xuất xóa. Vui lòng phê duyệt để loại bỏ quy tắc."
		actions = append(actions, entity.ProposedAction{
			ActionType:  "delete_spellcheck",
			Description: "Xóa quy tắc sửa chính tả cho từ gõ sai 'ipone' [MOCK LOCAL]",
			Params: map[string]interface{}{
				"ids":       []string{"mock-spellcheck-id-123"},
				"typo_word": "ipone",
			},
		})
	} else if strings.Contains(msg, "sản phẩm") || strings.Contains(msg, "kho") || strings.Contains(msg, "tồn") {
		// Fetch actual database products to demonstrate product search tool mock
		products, err := s.repo.GetAllProductsByTenantID(ctx, tenantID)
		if err == nil && len(products) > 0 {
			var pList []string
			for i, p := range products {
				pList = append(pList, fmt.Sprintf("- **%s** (%s): %s đồng, tồn kho: %d", p.Name, p.Brand, fmt.Sprintf("%.0f", p.Price), p.Inventory))
				if i >= 4 {
					break
				}
			}
			reply = "Dưới đây là một số sản phẩm tôi tìm thấy trong kho hàng của bạn:\n\n" + strings.Join(pList, "\n")
		} else {
			reply = "Hiện tại tôi không tìm thấy sản phẩm nào trong kho của cửa hàng này."
		}
	} else {
		reply = "Xin chào! Tôi là Trợ lý AI của Swift Search Engine. Tôi có thể giúp bạn tìm kiếm thông tin sản phẩm trong kho hoặc tạo các đề xuất điều chỉnh từ điển tìm kiếm (synonym, sửa lỗi chính tả). Bạn cần trợ giúp gì hôm nay?"
	}

	return reply, actions
}

// Global assistant tools schema declarations in OpenAI Format
var assistantTools = []map[string]interface{}{
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "search_products",
			"description": "Search products in the store's inventory by query or brand to check prices, stock level, status, etc.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Product name or brand search query (e.g., 'Akko', 'mouse', 'Logitech keyboard')",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of products to return (default 5)",
						"default":     5,
					},
				},
				"required": []string{"query"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "get_active_dictionaries",
			"description": "Get all active synonyms and spellcheck rules for the store.",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "create_synonym",
			"description": "Propose creating a new synonym rule between two words. The actual creation is performed only after the human administrator approves.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type":        "string",
						"description": "The main search term (e.g., 'máy tính')",
					},
					"synonym": map[string]interface{}{
						"type":        "string",
						"description": "The synonym term to map (e.g., 'computer')",
					},
					"is_bidirectional": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the synonym is bidirectional (e.g. keyword maps to synonym AND synonym maps to keyword)",
						"default":     true,
					},
				},
				"required": []string{"keyword", "synonym"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "delete_synonym",
			"description": "Propose deleting synonym rules between two words. The actual deletion is performed only after the human administrator approves.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type":        "string",
						"description": "The keyword term of the synonym rule to delete",
					},
					"synonym": map[string]interface{}{
						"type":        "string",
						"description": "The synonym term of the synonym rule to delete",
					},
				},
				"required": []string{"keyword", "synonym"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "create_spellcheck",
			"description": "Propose creating a new spelling correction (typo) rule. The actual creation is performed only after the human administrator approves.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"typo_word": map[string]interface{}{
						"type":        "string",
						"description": "The typo or misspelled word typed by the user (e.g., 'ipone')",
					},
					"correct_word": map[string]interface{}{
						"type":        "string",
						"description": "The correct word to replace it with (e.g., 'iPhone')",
					},
				},
				"required": []string{"typo_word", "correct_word"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "delete_spellcheck",
			"description": "Propose deleting spelling correction (typo) rules for a misspelled word. The actual deletion is performed only after the human administrator approves.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"typo_word": map[string]interface{}{
						"type":        "string",
						"description": "The typo word of the spellcheck rule to delete",
					},
					"correct_word": map[string]interface{}{
						"type":        "string",
						"description": "The correct word of the spellcheck rule to delete (optional)",
					},
				},
				"required": []string{"typo_word"},
			},
		},
	},
}
