package sut

import (
	"context"
	"testing"
)

func TestDestroyBeforeCreateIsNoop(t *testing.T) {
	adapter := &DockerAdapter{}
	if err := adapter.Destroy(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRejectsReuseAfterCreatedState(t *testing.T) {
	adapter := &DockerAdapter{name: "existing", created: true}
	if err := adapter.Create(context.Background(), CreateSpec{Name: "next"}); err == nil {
		t.Fatal("expected adapter reuse to be rejected")
	}
}
