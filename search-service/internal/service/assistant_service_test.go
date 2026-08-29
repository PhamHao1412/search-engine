package service_test

import (
	"context"
	"search-service/internal/entity"
	"search-service/internal/service"
	"strings"
	"testing"
)

func TestAssistantService_ChatWithAssistant_MockLocal_Help(t *testing.T) {
	repo := NewMockSearchRepository()
	// Initialize with empty apiKey to trigger mock response
	svc := service.NewAssistantService(repo, "", "")

	reply, proposedActions, _, err := svc.ChatWithAssistant(context.Background(), "tenant-1", "conv-1", "Chào trợ lý, bạn có thể làm được gì?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(proposedActions) != 0 {
		t.Errorf("expected 0 proposed actions for general conversation, got %d", len(proposedActions))
	}

	if !strings.Contains(reply, "Trợ lý AI") {
		t.Errorf("expected reply to mention assistant identity, got: %s", reply)
	}
}

func TestAssistantService_ChatWithAssistant_MockLocal_CreateSynonym(t *testing.T) {
	repo := NewMockSearchRepository()
	svc := service.NewAssistantService(repo, "", "")

	reply, proposedActions, _, err := svc.ChatWithAssistant(
		context.Background(),
		"tenant-1",
		"conv-1",
		"Thêm từ đồng nghĩa bàn phím cơ và phím cơ",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(proposedActions) != 1 {
		t.Fatalf("expected 1 proposed action, got %d", len(proposedActions))
	}

	act := proposedActions[0]
	if act.ActionType != "create_synonym" {
		t.Errorf("expected ActionType 'create_synonym', got '%s'", act.ActionType)
	}

	keyword := act.Params["keyword"].(string)
	synonym := act.Params["synonym"].(string)
	isBidirectional := act.Params["is_bidirectional"].(bool)

	if keyword != "bàn phím cơ" || synonym != "phím cơ" || !isBidirectional {
		t.Errorf("incorrect proposed action parameters: %v", act.Params)
	}

	if !strings.Contains(reply, "xác nhận") {
		t.Errorf("expected reply to mention confirmation request, got: %s", reply)
	}
}

func TestAssistantService_ChatWithAssistant_MockLocal_CreateSpellcheck(t *testing.T) {
	repo := NewMockSearchRepository()
	svc := service.NewAssistantService(repo, "", "")

	reply, proposedActions, _, err := svc.ChatWithAssistant(
		context.Background(),
		"tenant-1",
		"conv-1",
		"Tạo quy tắc sửa chính tả cho từ ako thành akko",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(proposedActions) != 1 {
		t.Fatalf("expected 1 proposed action, got %d", len(proposedActions))
	}

	act := proposedActions[0]
	if act.ActionType != "create_spellcheck" {
		t.Errorf("expected ActionType 'create_spellcheck', got '%s'", act.ActionType)
	}

	typo := act.Params["typo_word"].(string)
	correct := act.Params["correct_word"].(string)

	if typo != "ako" || correct != "akko" {
		t.Errorf("incorrect proposed spellcheck parameters: %v", act.Params)
	}

	if !strings.Contains(reply, "soạn sẵn") {
		t.Errorf("expected reply to contain test prompt elements, got: %s", reply)
	}
}

func TestAssistantService_ChatWithAssistant_MockLocal_ProductSearch(t *testing.T) {
	repo := NewMockSearchRepository()
	svc := service.NewAssistantService(repo, "", "")

	// Set up mock products in repo
	repo.GetAllProductsByTenantIDFn = func(ctx context.Context, tenantID string) ([]entity.Product, error) {
		return []entity.Product{
			{
				ID:        "prod-1",
				TenantID:  tenantID,
				Name:      "Bàn phím cơ Akko 3087 v2",
				Brand:     "Akko",
				Price:     1250000,
				Inventory: 15,
			},
			{
				ID:        "prod-2",
				TenantID:  tenantID,
				Name:      "Chuột gaming Logitech G102",
				Brand:     "Logitech",
				Price:     350000,
				Inventory: 4,
			},
		}, nil
	}

	reply, proposedActions, _, err := svc.ChatWithAssistant(
		context.Background(),
		"tenant-1",
		"conv-1",
		"Kiểm tra xem còn sản phẩm nào trong kho không?",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(proposedActions) != 0 {
		t.Errorf("expected 0 proposed actions for product lookup, got %d", len(proposedActions))
	}

	if !strings.Contains(reply, "Akko 3087") || !strings.Contains(reply, "Logitech G102") {
		t.Errorf("expected reply to list products, got: %s", reply)
	}
}
