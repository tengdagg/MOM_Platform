package biz

import "context"

type ChatChannelAdapter interface {
	Provider() string
	Start(ctx context.Context) error
	Stop() error
	Test(ctx context.Context) error
}
