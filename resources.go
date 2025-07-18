package rush

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed assets/images
var assetsFS embed.FS

// ResourceType 定义资源类型
type ResourceType string

const (
	ResourceSubmarine     ResourceType = "submarine"
	ResourceTitle         ResourceType = "title"
	ResourceGameOver      ResourceType = "gameover"
	ResourceWin           ResourceType = "win"
	ResourceCoin          ResourceType = "coin"
	ResourceBomb          ResourceType = "bomb"
	ResourceHandDrawnFont ResourceType = "handdrawn_font"
)

// ResourceManager 资源管理器
type ResourceManager struct {
	cache  map[ResourceType]*ebiten.Image
	mutex  sync.RWMutex
	loaded bool
}

// NewResourceManager 创建新的资源管理器
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		cache: make(map[ResourceType]*ebiten.Image),
	}
}

// LoadResource 加载单个资源
func (rm *ResourceManager) LoadResource(resourceType ResourceType) (*ebiten.Image, error) {
	rm.mutex.RLock()
	if img, exists := rm.cache[resourceType]; exists {
		rm.mutex.RUnlock()
		return img, nil
	}
	rm.mutex.RUnlock()

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// 双重检查，防止并发加载同一资源
	if img, exists := rm.cache[resourceType]; exists {
		return img, nil
	}

	// 获取资源路径
	path, err := rm.getResourcePath(resourceType)
	if err != nil {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}

	// 从嵌入文件系统读取资源
	b, err := assetsFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource %s: %w", resourceType, err)
	}

	// 创建图像
	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create image for %s: %w", resourceType, err)
	}

	// 缓存资源
	rm.cache[resourceType] = img
	log.Printf("Loaded resource: %s", resourceType)

	return img, nil
}

// LoadResourceSafe 安全加载资源，失败时返回nil
func (rm *ResourceManager) LoadResourceSafe(resourceType ResourceType) *ebiten.Image {
	img, err := rm.LoadResource(resourceType)
	if err != nil {
		log.Printf("Failed to load resource %s: %v", resourceType, err)
		return nil
	}
	return img
}

// PreloadResources 预加载所有资源
func (rm *ResourceManager) PreloadResources() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.loaded {
		return nil
	}

	resourceTypes := []ResourceType{
		ResourceSubmarine,
		ResourceTitle,
		ResourceGameOver,
		ResourceWin,
		ResourceCoin,
		ResourceBomb,
		ResourceHandDrawnFont,
	}

	for _, resourceType := range resourceTypes {
		path, err := rm.getResourcePath(resourceType)
		if err != nil {
			return fmt.Errorf("invalid resource type: %s", resourceType)
		}

		b, err := assetsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read resource %s: %w", resourceType, err)
		}

		img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("failed to create image for %s: %w", resourceType, err)
		}

		rm.cache[resourceType] = img
		log.Printf("Preloaded resource: %s", resourceType)
	}

	rm.loaded = true
	log.Println("All resources preloaded successfully")
	return nil
}

// GetResource 获取资源，如果未加载则自动加载
func (rm *ResourceManager) GetResource(resourceType ResourceType) *ebiten.Image {
	return rm.LoadResourceSafe(resourceType)
}

// IsResourceLoaded 检查资源是否已加载
func (rm *ResourceManager) IsResourceLoaded(resourceType ResourceType) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	_, exists := rm.cache[resourceType]
	return exists
}

// ClearCache 清空资源缓存
func (rm *ResourceManager) ClearCache() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.cache = make(map[ResourceType]*ebiten.Image)
	rm.loaded = false
	log.Println("Resource cache cleared")
}

// GetCacheSize 获取缓存大小
func (rm *ResourceManager) GetCacheSize() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return len(rm.cache)
}

// getResourcePath 获取资源路径
func (rm *ResourceManager) getResourcePath(resourceType ResourceType) (string, error) {
	switch resourceType {
	case ResourceSubmarine:
		return "assets/images/submarine.png", nil
	case ResourceTitle:
		return "assets/images/title.png", nil
	case ResourceGameOver:
		return "assets/images/gameover.png", nil
	case ResourceWin:
		return "assets/images/win.png", nil
	case ResourceCoin:
		return "assets/images/coin.png", nil
	case ResourceBomb:
		return "assets/images/bomb.png", nil
	case ResourceHandDrawnFont:
		return "assets/images/handdrawn_font.png", nil
	default:
		return "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// CreateFallbackImage 创建降级图像
func (rm *ResourceManager) CreateFallbackImage(width, height int, clr color.Color) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(clr)
	return img
}

// 全局资源管理器实例
var globalResourceManager *ResourceManager

// InitResourceManager 初始化全局资源管理器
func InitResourceManager() {
	globalResourceManager = NewResourceManager()
}

// GetResourceManager 获取全局资源管理器
func GetResourceManager() *ResourceManager {
	if globalResourceManager == nil {
		InitResourceManager()
	}
	return globalResourceManager
}
