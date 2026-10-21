package catalog

import (
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

var (
	publicSorts = []string{"relevance", "newest", "price_asc", "price_desc", "name_asc", "name_desc", "popular"}
	adminSorts  = []string{"newest", "oldest", "price_asc", "price_desc", "name_asc", "name_desc", "stock_asc", "popular"}
)

// ---------- Public ----------

func (s *Service) HandlePublicCategories(w http.ResponseWriter, r *http.Request) error {
	cats, err := s.ListCategories(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, cats)
	return nil
}

func (s *Service) HandlePublicCategory(w http.ResponseWriter, r *http.Request) error {
	c, err := s.CategoryBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, s.categoryDTO(c))
	return nil
}

func (s *Service) HandlePublicProducts(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	pq := ProductQuery{
		Published: true,
		Text:      q.String("q", 100),
		Featured:  q.OptBool("featured"),
		InStock:   q.OptBool("inStock"),
		MinPrice:  q.OptInt64("minPrice", 0, 100_000_000),
		MaxPrice:  q.OptInt64("maxPrice", 0, 100_000_000),
		Sort:      q.Enum("sort", "relevance", publicSorts...),
		Locale:    q.Enum("locale", "en", "en", "vi"),
		Page:      q.Page(24, 100),
	}
	slug := q.String("category", 120)
	if err := q.Err(); err != nil {
		return err
	}
	if slug != "" {
		c, err := s.CategoryBySlug(r.Context(), slug)
		if errors.Is(err, httpx.ErrNotFound) {
			httpx.Paged(w, []ProductDTO{}, pq.Page.Meta(0))
			return nil
		}
		if err != nil {
			return err
		}
		pq.CategoryID = &c.ID
	}
	items, meta, err := s.ListProducts(r.Context(), pq)
	if err != nil {
		return err
	}
	httpx.Paged(w, items, meta)
	return nil
}

func (s *Service) HandlePublicProduct(w http.ResponseWriter, r *http.Request) error {
	p, err := s.findProduct(r.Context(), bson.M{"slug": r.PathValue("slug"), "status": StatusPublished})
	if err != nil {
		return err
	}
	dto, err := s.productDTO(r.Context(), p)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleRecordView(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	ua := r.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512]
	}
	if err := s.RecordView(r.Context(), id, httpx.ClientIPFrom(r.Context()), ua); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// ---------- Admin: products ----------

func (s *Service) HandleAdminProducts(w http.ResponseWriter, r *http.Request) error {
	q := httpx.NewQuery(r)
	pq := ProductQuery{
		Text:       q.String("q", 100),
		Status:     q.Enum("status", "", StatusDraft, StatusPublished, StatusArchived),
		CategoryID: q.OptObjectID("category"),
		Featured:   q.OptBool("featured"),
		MinPrice:   q.OptInt64("minPrice", 0, 100_000_000),
		MaxPrice:   q.OptInt64("maxPrice", 0, 100_000_000),
		Sort:       q.Enum("sort", "newest", adminSorts...),
		Locale:     "en",
		Page:       q.Page(25, 100),
	}
	if u := q.OptBool("uncategorized"); u != nil {
		pq.Uncategorized = *u
	}
	if err := q.Err(); err != nil {
		return err
	}
	items, meta, err := s.ListProducts(r.Context(), pq)
	if err != nil {
		return err
	}
	httpx.Paged(w, items, meta)
	return nil
}

func (s *Service) HandleAdminProduct(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	p, err := s.findProduct(r.Context(), bson.M{"_id": id})
	if err != nil {
		return err
	}
	dto, err := s.productDTO(r.Context(), p)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleCreateProduct(w http.ResponseWriter, r *http.Request) error {
	var in ProductInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	dto, err := s.CreateProduct(r.Context(), actorID(r), &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, dto)
	return nil
}

func (s *Service) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in ProductInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	dto, err := s.UpdateProduct(r.Context(), actorID(r), id, &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	if err := s.DeleteProduct(r.Context(), id); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (s *Service) HandleAddProductImage(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	img, err := readImageUpload(w, r)
	if err != nil {
		return err
	}
	var alt *LocalizedText
	altEN, altVI := strings.TrimSpace(r.FormValue("altEn")), strings.TrimSpace(r.FormValue("altVi"))
	if len(altEN) > 200 || len(altVI) > 200 {
		return httpx.Validation(httpx.FieldError{Field: "alt", Message: "must be at most 200 characters"})
	}
	if altEN != "" || altVI != "" {
		alt = &LocalizedText{EN: altEN, VI: altVI}
	}
	dto, err := s.AddProductImage(r.Context(), actorID(r), id, img, alt)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, dto)
	return nil
}

func (s *Service) HandleRemoveProductImage(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	dto, err := s.RemoveProductImage(r.Context(), actorID(r), id, r.PathValue("imageId"))
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleReorderProductImages(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		ImageIDs []string `json:"imageIds" validate:"required,max=10,dive,max=32"`
	}
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	dto, err := s.ReorderProductImages(r.Context(), actorID(r), id, in.ImageIDs)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

// ---------- Admin: categories ----------

func (s *Service) HandleCreateCategory(w http.ResponseWriter, r *http.Request) error {
	var in CategoryInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	dto, err := s.CreateCategory(r.Context(), &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, dto)
	return nil
}

func (s *Service) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	var in CategoryInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	dto, err := s.UpdateCategory(r.Context(), id, &in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	if err := s.DeleteCategory(r.Context(), id); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (s *Service) HandleSetCategoryImage(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	img, err := readImageUpload(w, r)
	if err != nil {
		return err
	}
	dto, err := s.SetCategoryImage(r.Context(), id, img)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

func (s *Service) HandleRemoveCategoryImage(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.ObjectIDParam(r, "id")
	if err != nil {
		return err
	}
	dto, err := s.RemoveCategoryImage(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, dto)
	return nil
}

// ---------- helpers ----------

func actorID(r *http.Request) string {
	if p := auth.PrincipalFrom(r.Context()); p != nil {
		return p.UserID
	}
	return ""
}

// readImageUpload parses a multipart body with a single "file" part and runs it
// through image validation and re-encoding.
func readImageUpload(w http.ResponseWriter, r *http.Request) (*media.Processed, error) {
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+64<<10)
	// Small memory budget: anything larger spills to a temp file that the
	// standard library removes when the request ends.
	// The body is bounded by MaxBytesReader above.
	if err := r.ParseMultipartForm(2 << 20); err != nil { //nolint:gosec // G120: bounded above
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, media.ErrTooLarge
		}
		return nil, httpx.BadRequest("expected a multipart/form-data body")
	}
	files := r.MultipartForm.File["file"]
	if len(files) != 1 {
		return nil, httpx.Validation(httpx.FieldError{Field: "file", Message: "exactly one file is required"})
	}
	f, err := files[0].Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	img, err := media.Process(r.Context(), f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
