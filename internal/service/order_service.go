package service

import (
	"context"
	"encoding/json"
	"outbox-demo/internal/repository"
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(ctx context.Context, item string) error {
	payload, err := json.Marshal(map[string]string{"item": item})
	if err != nil {
		return err
	}
	return s.repo.CreateOrderWithEvent(ctx, item, "order.created", payload)
}
