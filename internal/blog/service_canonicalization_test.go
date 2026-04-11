package blog

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"

	"github.com/jourloy/somivyn/internal/storage"
)

type canonicalStorage struct {
	baseURL    string
	bucketName string
}

func (s *canonicalStorage) EnsureFolder(ctx context.Context, folder string) error { return nil }

func (s *canonicalStorage) Upload(ctx context.Context, folder, key string, body []byte, contentType string) (string, error) {
	return s.BuildPublicURL(folder, key), nil
}

func (s *canonicalStorage) BuildPublicURL(folder, key string) string {
	baseURL := strings.TrimSuffix(s.baseURL, "/")
	return baseURL + "/" + strings.Trim(s.bucketName, "/") + "/" + folder + "/" + strings.Trim(key, "/")
}

func (s *canonicalStorage) GetObject(ctx context.Context, folder, key string) (*storage.ObjectReader, error) {
	return nil, nil
}

func (s *canonicalStorage) HeadObject(ctx context.Context, folder, key string) (*storage.ObjectInfo, error) {
	return nil, nil
}

type stubRepository struct {
	createArticleInput *ArticleCreate
	updateArticleInput *ArticleUpdate

	createArticleResult  *Article
	updateArticleResult  *Article
	getArticleByIDResult *Article
	getArticleBySlug     *Article
	getArticlesResult    *ArticleListResponse
	tagsByArticleID      map[int64][]Tag
}

func (r *stubRepository) CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	return nil, nil
}

func (r *stubRepository) UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error) {
	return nil, nil
}

func (r *stubRepository) DeleteCategory(ctx context.Context, id int64) error { return nil }

func (r *stubRepository) GetCategoryById(ctx context.Context, id int64) (*Category, error) {
	return nil, nil
}

func (r *stubRepository) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	return nil, nil
}

func (r *stubRepository) GetAllCategories(ctx context.Context) ([]Category, error) { return nil, nil }

func (r *stubRepository) CreateTag(ctx context.Context, t TagCreate) (*Tag, error) { return nil, nil }

func (r *stubRepository) UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error) { return nil, nil }

func (r *stubRepository) DeleteTag(ctx context.Context, id int64) error { return nil }

func (r *stubRepository) GetTagById(ctx context.Context, id int64) (*Tag, error) { return nil, nil }

func (r *stubRepository) GetTagBySlug(ctx context.Context, slug string) (*Tag, error) {
	return nil, nil
}

func (r *stubRepository) GetAllTags(ctx context.Context) ([]Tag, error) { return nil, nil }

func (r *stubRepository) GetTagsByArticleId(ctx context.Context, articleId int64) ([]Tag, error) {
	if r.tagsByArticleID == nil {
		return []Tag{}, nil
	}
	return r.tagsByArticleID[articleId], nil
}

func (r *stubRepository) CreateArticle(ctx context.Context, a ArticleCreate) (*Article, error) {
	input := a
	r.createArticleInput = &input
	if r.createArticleResult != nil {
		cloned := *r.createArticleResult
		return &cloned, nil
	}

	return &Article{
		Id:              1,
		Slug:            a.Slug,
		TitleRu:         a.TitleRu,
		TitleEn:         a.TitleEn,
		ContentRu:       a.ContentRu,
		ContentEn:       a.ContentEn,
		PreviewImageUrl: a.PreviewImageUrl,
		OgImageUrl:      a.OgImageUrl,
		Status:          a.Status,
		CreatedAt:       time.Now(),
	}, nil
}

func (r *stubRepository) UpdateArticle(ctx context.Context, a ArticleUpdate) (*Article, error) {
	input := a
	r.updateArticleInput = &input
	if r.updateArticleResult != nil {
		cloned := *r.updateArticleResult
		return &cloned, nil
	}

	return &Article{
		Id:              a.Id,
		Slug:            a.Slug,
		TitleRu:         a.TitleRu,
		TitleEn:         a.TitleEn,
		ContentRu:       a.ContentRu,
		ContentEn:       a.ContentEn,
		PreviewImageUrl: a.PreviewImageUrl,
		OgImageUrl:      a.OgImageUrl,
		Status:          a.Status,
		CreatedAt:       time.Now(),
	}, nil
}

func (r *stubRepository) DeleteArticle(ctx context.Context, id int64) error { return nil }

func (r *stubRepository) GetArticleById(ctx context.Context, id int64) (*Article, error) {
	if r.getArticleByIDResult == nil {
		return &Article{Id: id, Status: "draft", CreatedAt: time.Now()}, nil
	}
	cloned := *r.getArticleByIDResult
	return &cloned, nil
}

func (r *stubRepository) GetArticleBySlug(ctx context.Context, slug string) (*Article, error) {
	if r.getArticleBySlug == nil {
		return nil, nil
	}
	cloned := *r.getArticleBySlug
	return &cloned, nil
}

func (r *stubRepository) ArticleSlugExists(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

func (r *stubRepository) GetArticles(ctx context.Context, params ArticleListParams, includeAll bool) (*ArticleListResponse, error) {
	if r.getArticlesResult == nil {
		return &ArticleListResponse{}, nil
	}
	cloned := *r.getArticlesResult
	cloned.Articles = append([]Article(nil), r.getArticlesResult.Articles...)
	return &cloned, nil
}

func (r *stubRepository) IncrementViewCount(ctx context.Context, id int64) error { return nil }

func (r *stubRepository) UpdateReadingTime(ctx context.Context, id int64, minutes int) error {
	return nil
}

func (r *stubRepository) SetPublishedAt(ctx context.Context, id int64) error { return nil }

func (r *stubRepository) SetArticleTags(ctx context.Context, articleId int64, tagIds []int64) error {
	return nil
}

func (r *stubRepository) CreateFeedback(ctx context.Context, f FeedbackCreate) (*Feedback, error) {
	return nil, nil
}

func (r *stubRepository) GetFeedbackByUser(ctx context.Context, articleId int64, userId string) (*Feedback, error) {
	return nil, nil
}

func (r *stubRepository) GetFeedbackBySession(ctx context.Context, articleId int64, sessionId string) (*Feedback, error) {
	return nil, nil
}

func (r *stubRepository) GetFeedbackStats(ctx context.Context, articleId int64) (*FeedbackStats, error) {
	return nil, nil
}

func newTestService(t *testing.T, repo *stubRepository) *service {
	t.Helper()

	storageFake := &canonicalStorage{
		baseURL:    "https://cdn.example.com/storage",
		bucketName: "somivyn-images",
	}
	urlCanonicalizer, err := storage.NewBlogImageURLCanonicalizerFromService(storageFake)
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizerFromService() error = %v", err)
	}
	imageURLMapper, err := newBlogImageURLMapper(urlCanonicalizer, "https://api.example.com")
	if err != nil {
		t.Fatalf("newBlogImageURLMapper() error = %v", err)
	}

	return &service{
		repo:             repo,
		storage:          storageFake,
		logger:           log.NewWithOptions(io.Discard, log.Options{}),
		urlCanonicalizer: urlCanonicalizer,
		imageURLMapper:   imageURLMapper,
	}
}

func TestCreateArticleCanonicalizesBlogImageFieldsBeforeSaving(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	svc := newTestService(t, repo)

	previewImageURL := "/somivyn-blog-images/2026/03/preview.png"
	ogImageURL := "https://api.somivyn.com/somivyn-blog-images/2026/03/og.png"
	article, err := svc.CreateArticle(context.Background(), ArticleCreate{
		Slug:            "my-article",
		TitleRu:         "RU",
		TitleEn:         "EN",
		ContentRu:       `<p><img src="/somivyn-blog-images/2026/03/body.png" /></p>`,
		ContentEn:       `<p><img src="https://somivyn.com/somivyn-blog-images/2026/03/body-en.png" /></p>`,
		PreviewImageUrl: &previewImageURL,
		OgImageUrl:      &ogImageURL,
		Status:          "draft",
	})
	if err != nil {
		t.Fatalf("CreateArticle() error = %v", err)
	}

	if repo.createArticleInput == nil {
		t.Fatalf("expected repository create input to be captured")
	}

	if got := *repo.createArticleInput.PreviewImageUrl; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/preview.png" {
		t.Fatalf("previewImageUrl = %q", got)
	}
	if got := *repo.createArticleInput.OgImageUrl; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/og.png" {
		t.Fatalf("ogImageUrl = %q", got)
	}
	if got := repo.createArticleInput.ContentRu; got != `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/body.png" /></p>` {
		t.Fatalf("contentRu = %q", got)
	}
	if got := repo.createArticleInput.ContentEn; got != `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/body-en.png" /></p>` {
		t.Fatalf("contentEn = %q", got)
	}
	if article.PreviewImageUrl == nil || *article.PreviewImageUrl != "https://api.example.com/api/v1/blog/images/2026/03/preview.png" {
		t.Fatalf("response previewImageUrl = %#v", article.PreviewImageUrl)
	}
}

func TestUpdateArticleCanonicalizesBlogImageFieldsBeforeSaving(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		getArticleByIDResult: &Article{
			Id:        7,
			Status:    "draft",
			CreatedAt: time.Now(),
		},
	}
	svc := newTestService(t, repo)

	previewImageURL := "https://minio.jourloy.com/somivyn-blog-images/2026/03/preview.png"
	ogImageURL := "https://s3.somivyn.com/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/blog/2026/03/og.png"
	_, err := svc.UpdateArticle(context.Background(), ArticleUpdate{
		Id:              7,
		Slug:            "my-article",
		TitleRu:         "RU",
		TitleEn:         "EN",
		ContentRu:       `<p><img src="/somivyn-blog-images/2026/03/body.png" /></p>`,
		ContentEn:       `<p><img src="https://api.somivyn.com/somivyn-blog-images/2026/03/body-en.png" /></p>`,
		PreviewImageUrl: &previewImageURL,
		OgImageUrl:      &ogImageURL,
		Status:          "public",
	})
	if err != nil {
		t.Fatalf("UpdateArticle() error = %v", err)
	}

	if repo.updateArticleInput == nil {
		t.Fatalf("expected repository update input to be captured")
	}

	if got := *repo.updateArticleInput.PreviewImageUrl; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/preview.png" {
		t.Fatalf("previewImageUrl = %q", got)
	}
	if got := *repo.updateArticleInput.OgImageUrl; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/og.png" {
		t.Fatalf("ogImageUrl = %q", got)
	}
}

func TestGetArticleByIdCanonicalizesAdminResponse(t *testing.T) {
	t.Parallel()

	previewImageURL := "https://api.somivyn.com/somivyn-blog-images/2026/03/preview.png"
	repo := &stubRepository{
		getArticleByIDResult: &Article{
			Id:              9,
			Slug:            "admin-article",
			TitleRu:         "RU",
			TitleEn:         "EN",
			ContentRu:       `<p><img src="/somivyn-blog-images/2026/03/body.png" /></p>`,
			ContentEn:       `<p>EN</p>`,
			PreviewImageUrl: &previewImageURL,
			Status:          "draft",
			CreatedAt:       time.Now(),
		},
	}
	svc := newTestService(t, repo)

	article, err := svc.GetArticleById(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetArticleById() error = %v", err)
	}

	if article.PreviewImageUrl == nil || *article.PreviewImageUrl != "https://api.example.com/api/v1/blog/images/2026/03/preview.png" {
		t.Fatalf("previewImageUrl = %#v", article.PreviewImageUrl)
	}
	if article.ContentRu != `<p><img src="https://api.example.com/api/v1/blog/images/2026/03/body.png" /></p>` {
		t.Fatalf("contentRu = %q", article.ContentRu)
	}
}

func TestGetAllArticlesCanonicalizesAdminListResponse(t *testing.T) {
	t.Parallel()

	previewImageURL := "https://somivyn.com/somivyn-blog-images/2026/03/preview.png"
	repo := &stubRepository{
		getArticlesResult: &ArticleListResponse{
			Articles: []Article{
				{
					Id:              3,
					Slug:            "list-article",
					TitleRu:         "RU",
					TitleEn:         "EN",
					ContentRu:       `<p><img src="/somivyn-blog-images/2026/03/body.png" /></p>`,
					ContentEn:       `<p>EN</p>`,
					PreviewImageUrl: &previewImageURL,
					Status:          "draft",
					CreatedAt:       time.Now(),
				},
			},
			Total:      1,
			Page:       1,
			PerPage:    10,
			TotalPages: 1,
		},
	}
	svc := newTestService(t, repo)

	response, err := svc.GetAllArticles(context.Background(), ArticleListParams{})
	if err != nil {
		t.Fatalf("GetAllArticles() error = %v", err)
	}

	if got := *response.Articles[0].PreviewImageUrl; got != "https://api.example.com/api/v1/blog/images/2026/03/preview.png" {
		t.Fatalf("previewImageUrl = %q", got)
	}
}

func TestGetPublicArticleBySlugCanonicalizesPublicResponse(t *testing.T) {
	t.Parallel()

	previewImageURL := "https://minio.jourloy.com/somivyn-blog-images/2026/03/preview.png"
	repo := &stubRepository{
		getArticleBySlug: &Article{
			Id:                 11,
			Slug:               "public-article",
			TitleRu:            "RU",
			TitleEn:            "EN",
			ContentRu:          `<p><img src="/somivyn-blog-images/2026/03/body.png" /></p>`,
			ContentEn:          `<p>EN</p>`,
			PreviewImageUrl:    &previewImageURL,
			Status:             "public",
			ReadingTimeMinutes: 3,
			CreatedAt:          time.Now(),
		},
	}
	svc := newTestService(t, repo)

	article, err := svc.GetPublicArticleBySlug(context.Background(), "public-article", ViewerAccess{})
	if err != nil {
		t.Fatalf("GetPublicArticleBySlug() error = %v", err)
	}

	if article.PreviewImageUrl == nil || *article.PreviewImageUrl != "https://api.example.com/api/v1/blog/images/2026/03/preview.png" {
		t.Fatalf("previewImageUrl = %#v", article.PreviewImageUrl)
	}
	if article.ContentRu != `<p><img src="https://api.example.com/api/v1/blog/images/2026/03/body.png" /></p>` {
		t.Fatalf("contentRu = %q", article.ContentRu)
	}
}
