package imageworker

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	imagingv1 "github.com/atcreations/atcreations/apps/api/internal/gen/imaging/v1"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/media/mediatest"
)

const token = "0123456789abcdef0123456789abcdef"

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{R: 120, G: 80, B: 200, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func start(t *testing.T, store *mediatest.Store) imagingv1.ImageWorkerServiceClient {
	t.Helper()
	return startWith(t, New(store, 1))
}

func startWith(t *testing.T, worker *Server) imagingv1.ImageWorkerServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	unary, stream := AuthInterceptors(token)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(RecoverUnary, unary), grpc.ChainStreamInterceptor(RecoverStream, stream))
	imagingv1.RegisterImageWorkerServiceServer(srv, worker)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return imagingv1.NewImageWorkerServiceClient(conn)
}

func authed() context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
}

// run collects the stream: the stages seen and the result or error.
func run(t *testing.T, c imagingv1.ImageWorkerServiceClient, ctx context.Context, req *imagingv1.ProcessImageRequest) ([]imagingv1.Stage, *imagingv1.Result, error) {
	t.Helper()
	stream, err := c.ProcessImage(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	var stages []imagingv1.Stage
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stages, nil, nil
		}
		if err != nil {
			return stages, nil, err
		}
		if p := ev.GetProgress(); p != nil {
			stages = append(stages, p.GetStage())
		}
		if r := ev.GetResult(); r != nil {
			return stages, r, nil
		}
	}
}

func req(src, dst string) *imagingv1.ProcessImageRequest {
	return &imagingv1.ProcessImageRequest{JobId: "j1", SourceKey: src, OutputKey: dst, MaxEdge: 100, Quality: 80}
}

func TestProcessImageSuccess(t *testing.T) {
	store := mediatest.New()
	_ = store.Put(context.Background(), "uploads/products/p1/a.png", pngBytes(t, 400, 200), "image/png")
	c := start(t, store)

	stages, res, err := run(t, c, authed(), req("uploads/products/p1/a.png", "products/p1/j1.webp"))
	if err != nil {
		t.Fatal(err)
	}
	if res.GetWidth() != 100 || res.GetHeight() != 50 || res.GetContentType() != "image/webp" {
		t.Fatalf("result = %+v", res)
	}
	if !store.Has("products/p1/j1.webp") {
		t.Fatal("output not stored")
	}
	if store.Has("uploads/products/p1/a.png") {
		t.Fatal("original must be deleted after success")
	}
	want := []imagingv1.Stage{imagingv1.Stage_STAGE_WAITING, imagingv1.Stage_STAGE_DOWNLOADING, imagingv1.Stage_STAGE_DECODING,
		imagingv1.Stage_STAGE_RESIZING, imagingv1.Stage_STAGE_ENCODING, imagingv1.Stage_STAGE_UPLOADING}
	if len(stages) != len(want) {
		t.Fatalf("stages = %v", stages)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Fatalf("stages = %v, want %v", stages, want)
		}
	}

	// Retrying after a lost response reports the existing output instead of failing.
	_, res2, err := run(t, c, authed(), req("uploads/products/p1/a.png", "products/p1/j1.webp"))
	if err != nil || res2.GetWidth() != 100 || res2.GetHeight() != 50 {
		t.Fatalf("idempotent retry: %+v %v", res2, err)
	}
}

func TestProcessImageInvalidDeletesOriginal(t *testing.T) {
	store := mediatest.New()
	_ = store.Put(context.Background(), "uploads/products/p1/bad.png", []byte("<?php echo 1; ?>"), "image/png")
	c := start(t, store)
	_, _, err := run(t, c, authed(), req("uploads/products/p1/bad.png", "products/p1/j1.webp"))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v, want InvalidArgument", err)
	}
	if store.Has("uploads/products/p1/bad.png") || store.Has("products/p1/j1.webp") {
		t.Fatal("invalid original must be deleted and nothing written")
	}
}

func TestProcessImageMissing(t *testing.T) {
	c := start(t, mediatest.New())
	_, _, err := run(t, c, authed(), req("uploads/products/p1/gone.png", "products/p1/j1.webp"))
	if status.Code(err) != codes.NotFound {
		t.Fatalf("err = %v, want NotFound", err)
	}
}

func TestProcessImageTransientStorageError(t *testing.T) {
	store := mediatest.New()
	_ = store.Put(context.Background(), "uploads/products/p1/a.png", pngBytes(t, 50, 50), "image/png")
	store.FailPuts = 1 // the output write is throttled
	c := start(t, store)
	_, _, err := run(t, c, authed(), req("uploads/products/p1/a.png", "products/p1/j1.webp"))
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v, want Unavailable", err)
	}
	if !store.Has("uploads/products/p1/a.png") {
		t.Fatal("the original must survive a transient failure so the job can be retried")
	}
}

func TestProcessImageRejectsBadRequests(t *testing.T) {
	c := start(t, mediatest.New())
	for name, r := range map[string]*imagingv1.ProcessImageRequest{
		"source outside uploads": req("products/p1/a.png", "products/p1/j1.webp"),
		"output inside uploads":  req("uploads/a.png", "uploads/b.webp"),
		"traversal":              req("uploads/../products/x.png", "products/p1/j1.webp"),
		"not webp":               req("uploads/a.png", "products/p1/j1.png"),
		"edge too large":         {SourceKey: "uploads/a.png", OutputKey: "products/a.webp", MaxEdge: 10000, Quality: 80},
	} {
		if _, _, err := run(t, c, authed(), r); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: err = %v, want InvalidArgument", name, err)
		}
	}
}

func TestProcessImageRequiresToken(t *testing.T) {
	c := start(t, mediatest.New())
	_, _, err := run(t, c, context.Background(), req("uploads/a.png", "products/a.webp"))
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("no token: err = %v", err)
	}
	bad := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer wrong")
	if _, _, err := run(t, c, bad, req("uploads/a.png", "products/a.webp")); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong token: err = %v", err)
	}
}

func TestProcessImageReportsErrorCode(t *testing.T) {
	store := mediatest.New()
	damaged := pngBytes(t, 200, 200)
	_ = store.Put(context.Background(), "uploads/products/p1/cut.png", damaged[:len(damaged)/2], "image/png")
	c := start(t, store)
	_, _, err := run(t, c, authed(), req("uploads/products/p1/cut.png", "products/p1/j1.webp"))
	st := status.Convert(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("err = %v, want InvalidArgument", err)
	}
	var reason string
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			reason = info.GetReason()
		}
	}
	if reason != media.CodeUnreadable {
		t.Fatalf("reason = %q, want %q", reason, media.CodeUnreadable)
	}
	if store.Has("uploads/products/p1/cut.png") {
		t.Fatal("damaged original must be deleted")
	}
}

func TestProcessImageWritesVariants(t *testing.T) {
	store := mediatest.New()
	_ = store.Put(context.Background(), "uploads/products/p1/a.png", pngBytes(t, 800, 400), "image/png")
	c := start(t, store)
	r := req("uploads/products/p1/a.png", "products/p1/j1.webp")
	r.MaxEdge, r.VariantEdges = 600, []uint32{100, 300, 900}
	_, res, err := run(t, c, authed(), r)
	if err != nil {
		t.Fatal(err)
	}
	// 900 is not below the 600 px result, so it is skipped.
	if len(res.GetVariants()) != 2 || res.GetVariants()[0].GetWidth() != 100 || res.GetVariants()[1].GetKey() != "products/p1/j1-300.webp" {
		t.Fatalf("variants = %v", res.GetVariants())
	}
	for _, k := range []string{"products/p1/j1.webp", "products/p1/j1-100.webp", "products/p1/j1-300.webp"} {
		if !store.Has(k) {
			t.Fatalf("%s not written", k)
		}
	}
	// A retry after a lost response reports the variants too.
	_, again, err := run(t, c, authed(), r)
	if err != nil || len(again.GetVariants()) != 2 {
		t.Fatalf("retry: %v %v", again.GetVariants(), err)
	}

	// Variants for an existing image: nothing is deleted or re-encoded.
	_ = store.Put(context.Background(), "products/p1/old.webp", pngBytes(t, 500, 500), "image/webp")
	only := &imagingv1.ProcessImageRequest{JobId: "b1", SourceKey: "products/p1/old.webp", OutputKey: "products/p1/old.webp",
		MaxEdge: 2400, Quality: 80, VariantEdges: []uint32{400}, VariantsOnly: true}
	_, res, err = run(t, c, authed(), only)
	if err != nil || len(res.GetVariants()) != 1 || !store.Has("products/p1/old.webp") || !store.Has("products/p1/old-400.webp") {
		t.Fatalf("variants only: %v %v", res.GetVariants(), err)
	}
	only.SourceKey = "uploads/x.png"
	if _, _, err := run(t, c, authed(), only); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("variants_only with a different source: %v", err)
	}
}
