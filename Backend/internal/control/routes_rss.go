package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxRSSFeedBytes = 10 * 1024 * 1024

type createRSSSubscriptionRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type rssFetchResult struct {
	Subscription common.RSSSubscription `json:"subscription"`
	Items        []common.RSSItem       `json:"items"`
	Created      int                    `json:"created"`
	Updated      int                    `json:"updated"`
}

type parsedFeed struct {
	Title       string
	Description string
	Link        string
	Items       []parsedFeedItem
}

type parsedFeedItem struct {
	GUID        string
	Title       string
	Link        string
	Author      string
	Summary     string
	Content     string
	PublishedAt *time.Time
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
	DCreator    string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	PubDate     string `xml:"pubDate"`
	Updated     string `xml:"updated"`
}

type atomDocument struct {
	Title    string      `xml:"title"`
	SubTitle string      `xml:"subtitle"`
	Links    []atomLink  `xml:"link"`
	Entries  []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomText struct {
	Body string `xml:",chardata"`
}

type atomEntry struct {
	ID        string     `xml:"id"`
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   atomText   `xml:"summary"`
	Content   atomText   `xml:"content"`
	Author    atomAuthor `xml:"author"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

func (a *App) createRSSSubscription(c *fiber.Ctx) error {
	var req createRSSSubscriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	feedURL, err := normalizeRSSURL(req.URL)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	uid, _ := c.Locals("uid").(string)
	var sub common.RSSSubscription
	err = a.db.Where("url = ?", feedURL).First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now()
		sub = common.RSSSubscription{
			ID:         uuid.NewString(),
			URL:        feedURL,
			Title:      strings.TrimSpace(req.Title),
			StorageDir: filepath.Join(a.cfg.RSSRoot, uuid.NewString()),
			CreatedBy:  uid,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := a.db.Create(&sub).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	} else if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result, err := a.fetchRSSSubscription(c.UserContext(), &sub)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (a *App) listRSSSubscriptions(c *fiber.Ctx) error {
	var subs []common.RSSSubscription
	query := a.db.Order("created_at desc")
	role, _ := c.Locals("role").(string)
	if !isPrivilegedRole(role) {
		uid, _ := c.Locals("uid").(string)
		query = query.Where("created_by = ?", uid)
	}
	if err := query.Find(&subs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(subs)
}

func (a *App) getRSSSubscription(c *fiber.Ctx) error {
	sub, err := a.loadAccessibleRSSSubscription(c)
	if err != nil {
		return err
	}
	return c.JSON(sub)
}

func (a *App) refreshRSSSubscription(c *fiber.Ctx) error {
	sub, err := a.loadAccessibleRSSSubscription(c)
	if err != nil {
		return err
	}
	result, err := a.fetchRSSSubscription(c.UserContext(), &sub)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(result)
}

func (a *App) listRSSItems(c *fiber.Ctx) error {
	sub, err := a.loadAccessibleRSSSubscription(c)
	if err != nil {
		return err
	}
	var items []common.RSSItem
	if err := a.db.Where("subscription_id = ?", sub.ID).Order("fetched_at desc").Find(&items).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(items)
}

func (a *App) deleteRSSSubscription(c *fiber.Ctx) error {
	sub, err := a.loadAccessibleRSSSubscription(c)
	if err != nil {
		return err
	}
	tx := a.db.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	if err := tx.Where("subscription_id = ?", sub.ID).Delete(&common.RSSItem{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Where("id = ?", sub.ID).Delete(&common.RSSSubscription{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if strings.TrimSpace(sub.StorageDir) != "" {
		_ = os.RemoveAll(sub.StorageDir)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) loadAccessibleRSSSubscription(c *fiber.Ctx) (common.RSSSubscription, error) {
	var sub common.RSSSubscription
	if err := a.db.Where("id = ?", strings.TrimSpace(c.Params("id"))).First(&sub).Error; err != nil {
		return sub, fiber.NewError(fiber.StatusNotFound, "rss subscription not found")
	}
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	if !isPrivilegedRole(role) && sub.CreatedBy != uid {
		return sub, fiber.NewError(fiber.StatusNotFound, "rss subscription not found")
	}
	return sub, nil
}

func (a *App) fetchRSSSubscription(ctx context.Context, sub *common.RSSSubscription) (rssFetchResult, error) {
	data, err := fetchRSSFeed(ctx, sub.URL)
	if err != nil {
		return rssFetchResult{}, err
	}
	feed, err := parseRSSFeed(data)
	if err != nil {
		return rssFetchResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(sub.StorageDir, "items"), 0o755); err != nil {
		return rssFetchResult{}, err
	}
	if err := os.WriteFile(filepath.Join(sub.StorageDir, "feed.xml"), data, 0o644); err != nil {
		return rssFetchResult{}, err
	}

	now := time.Now()
	if strings.TrimSpace(feed.Title) != "" {
		sub.Title = feed.Title
	}
	sub.Description = feed.Description
	sub.Link = feed.Link
	sub.LastFetchedAt = &now
	sub.UpdatedAt = now
	if err := a.db.Save(sub).Error; err != nil {
		return rssFetchResult{}, err
	}

	result := rssFetchResult{Subscription: *sub}
	for _, parsed := range feed.Items {
		item, created, err := a.upsertRSSItem(*sub, parsed, now)
		if err != nil {
			return rssFetchResult{}, err
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (a *App) upsertRSSItem(sub common.RSSSubscription, parsed parsedFeedItem, fetchedAt time.Time) (common.RSSItem, bool, error) {
	guid := stableRSSItemGUID(parsed)
	item := common.RSSItem{}
	err := a.db.Where("subscription_id = ? AND guid = ?", sub.ID, guid).First(&item).Error
	created := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		item = common.RSSItem{
			ID:             uuid.NewString(),
			SubscriptionID: sub.ID,
			GUID:           guid,
			ContentPath:    filepath.Join(sub.StorageDir, "items", guid+".json"),
		}
		created = true
	} else if err != nil {
		return item, false, err
	}

	item.Title = parsed.Title
	item.Link = parsed.Link
	item.Author = parsed.Author
	item.Summary = parsed.Summary
	item.Content = parsed.Content
	item.PublishedAt = parsed.PublishedAt
	item.FetchedAt = fetchedAt

	if err := writeRSSItemFile(item); err != nil {
		return item, false, err
	}
	if created {
		return item, true, a.db.Create(&item).Error
	}
	return item, false, a.db.Save(&item).Error
}

func fetchRSSFeed(ctx context.Context, feedURL string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Araneae RSS Fetcher/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rss feed returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRSSFeedBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxRSSFeedBytes {
		return nil, fmt.Errorf("rss feed is too large (max %d bytes)", maxRSSFeedBytes)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errors.New("rss feed is empty")
	}
	return data, nil
}

func normalizeRSSURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("url is required")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("url must be an absolute http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("url must use http or https")
	}
	if parsed.User != nil {
		return "", errors.New("url must not contain credentials")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func parseRSSFeed(data []byte) (parsedFeed, error) {
	var probe struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &probe); err != nil {
		return parsedFeed{}, fmt.Errorf("parse rss xml failed: %w", err)
	}

	switch strings.ToLower(probe.XMLName.Local) {
	case "rss", "rdf":
		var doc rssDocument
		if err := xml.Unmarshal(data, &doc); err != nil {
			return parsedFeed{}, err
		}
		return convertRSSDocument(doc), nil
	case "feed":
		var doc atomDocument
		if err := xml.Unmarshal(data, &doc); err != nil {
			return parsedFeed{}, err
		}
		return convertAtomDocument(doc), nil
	default:
		return parsedFeed{}, fmt.Errorf("unsupported feed root %q", probe.XMLName.Local)
	}
}

func convertRSSDocument(doc rssDocument) parsedFeed {
	feed := parsedFeed{
		Title:       cleanFeedText(doc.Channel.Title),
		Description: cleanFeedText(doc.Channel.Description),
		Link:        strings.TrimSpace(doc.Channel.Link),
	}
	for _, raw := range doc.Channel.Items {
		content := firstNonEmpty(raw.Content, raw.Description)
		published := parseFeedTime(firstNonEmpty(raw.PubDate, raw.Updated))
		feed.Items = append(feed.Items, parsedFeedItem{
			GUID:        firstNonEmpty(raw.GUID, raw.Link, raw.Title),
			Title:       cleanFeedText(raw.Title),
			Link:        strings.TrimSpace(raw.Link),
			Author:      cleanFeedText(firstNonEmpty(raw.DCreator, raw.Author)),
			Summary:     cleanFeedText(raw.Description),
			Content:     strings.TrimSpace(content),
			PublishedAt: published,
		})
	}
	return feed
}

func convertAtomDocument(doc atomDocument) parsedFeed {
	feed := parsedFeed{
		Title:       cleanFeedText(doc.Title),
		Description: cleanFeedText(doc.SubTitle),
		Link:        atomBestLink(doc.Links),
	}
	for _, raw := range doc.Entries {
		content := firstNonEmpty(raw.Content.Body, raw.Summary.Body)
		published := parseFeedTime(firstNonEmpty(raw.Published, raw.Updated))
		link := atomBestLink(raw.Links)
		feed.Items = append(feed.Items, parsedFeedItem{
			GUID:        firstNonEmpty(raw.ID, link, raw.Title),
			Title:       cleanFeedText(raw.Title),
			Link:        link,
			Author:      cleanFeedText(raw.Author.Name),
			Summary:     cleanFeedText(raw.Summary.Body),
			Content:     strings.TrimSpace(content),
			PublishedAt: published,
		})
	}
	return feed
}

func atomBestLink(links []atomLink) string {
	for _, link := range links {
		if strings.TrimSpace(link.Rel) == "" || link.Rel == "alternate" {
			return strings.TrimSpace(link.Href)
		}
	}
	if len(links) > 0 {
		return strings.TrimSpace(links[0].Href)
	}
	return ""
}

func parseFeedTime(raw string) *time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t
		}
	}
	return nil
}

func stableRSSItemGUID(item parsedFeedItem) string {
	identity := firstNonEmpty(item.GUID, item.Link, item.Title)
	sum := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(sum[:])
}

func writeRSSItemFile(item common.RSSItem) error {
	if err := os.MkdirAll(filepath.Dir(item.ContentPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(item.ContentPath, data, 0o644)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cleanFeedText(value string) string {
	return strings.TrimSpace(value)
}
