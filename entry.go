package javadocr

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/lukegb/javadocr/maven"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SnapshotExpiryWindow = 1 * time.Minute
	GCInterval           = SnapshotExpiryWindow / 2
	LruCacheSize         = 512 * 1024 * 1024
)

type JavadocCache map[maven.Coordinate]*JavadocCached

type JavadocCacheKeys struct {
	c  []maven.Coordinate
	jc JavadocCache
}

func (jck *JavadocCacheKeys) Len() int {
	return len(jck.c)
}

func (jck *JavadocCacheKeys) Less(i, j int) bool {
	return jck.jc[jck.c[i]].accessed.After(jck.jc[jck.c[i]].accessed)
}

func (jck *JavadocCacheKeys) Swap(i, j int) {
	jck.c[i], jck.c[j] = jck.c[j], jck.c[i]
}

type JavadocHandler struct {
	repository maven.Repository
	coordinate maven.Coordinate

	compat map[string]bool

	versions        []maven.Coordinate
	excludeVersions map[string]bool
	versionsLock    sync.RWMutex

	versionCache     JavadocCache
	versionCacheLock sync.RWMutex
}

type JavadocCached struct {
	server   *ZipFileSystem
	artifact *maven.Artifact
	size     int64
	cached   time.Time
	accessed time.Time
}

func (h *JavadocHandler) refresher() {
	// yay
	for {
		(func() {
			log.Println("Checking for new versions")

			err := h.populateVersions()
			log.Printf("New versions check concluded with result %v", err)
		})()

		(func() {
			h.versionCacheLock.Lock()
			defer h.versionCacheLock.Unlock()
			log.Println("Checking SNAPSHOT artifacts for expiry")
			nvc := make(JavadocCache)
			for c, el := range h.versionCache {
				if !c.IsSnapshot() {
					nvc[c] = el
					continue
				}

				if el.cached.After(time.Now().Add(-SnapshotExpiryWindow)) {
					nvc[c] = el
					continue
				}

				log.Printf("Expiring %v", el.artifact.Coordinate.String())
			}
			h.versionCache = nvc
		})()
		time.Sleep(GCInterval)
	}
}

func (h *JavadocHandler) ExcludeVersion(v string) {
	h.versionsLock.Lock()
	defer h.versionsLock.Unlock()
	h.excludeVersions[v] = true
}

func (h *JavadocHandler) IncludeVersion(v string) {
	h.versionsLock.Lock()
	defer h.versionsLock.Unlock()
	delete(h.excludeVersions, v)
}

// must be called whilst holding versionCacheLock!
func (h *JavadocHandler) tidyVersionCache() {
	jcks := new(JavadocCacheKeys)
	jcks.c = make([]maven.Coordinate, 0, len(h.versionCache))
	jcks.jc = h.versionCache
	for jck := range h.versionCache {
		jcks.c = append(jcks.c, jck)
	}

	sort.Sort(jcks)

	var runningTotalSize int64 = 0
	cutoff := -1
	for k, n := range jcks.c {
		if cutoff != -1 {
			delete(h.versionCache, n)
		}

		runningTotalSize += h.versionCache[n].size
		if runningTotalSize > LruCacheSize {
			cutoff = k
			break
		}
	}

	if cutoff > 0 {
		log.Printf("Culled LRU cache at point %d", h.versionCache)
	}

}

func (h *JavadocHandler) calculateValidUntil(c maven.Coordinate, cachedAt time.Time) time.Time {
	if c.IsSnapshot() {
		return cachedAt.Add(SnapshotExpiryWindow)
	} else {
		// 30 days
		return time.Now().Add(2592000 * time.Second)
	}
}

func (h *JavadocHandler) fetchForCoordinate(c maven.Coordinate) (*ZipFileSystem, time.Time, error) {
	jc, ok := (func() (*JavadocCached, bool) {
		h.versionCacheLock.RLock()
		defer h.versionCacheLock.RUnlock()

		jc, ok := h.versionCache[c]

		if ok {
			validUntil := h.calculateValidUntil(c, jc.cached)
			now := time.Now()
			if validUntil.Before(now) || validUntil.Equal(now) {
				// NOPE NOT VALID
				return jc, false
			}
		}

		return jc, ok
	})()
	if ok {
		return jc.server, h.calculateValidUntil(c, jc.cached), nil
	}

	artifact, err := h.repository.Resolve(c)
	if err != nil {
		return nil, time.Now(), err
	}

	rc, err := artifact.Fetch()
	defer rc.Close()
	if err != nil {
		return nil, time.Now(), err
	}

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, time.Now(), err
	}

	bb := bytes.NewReader(data)
	zr, err := zip.NewReader(bb, int64(len(data)))
	if err != nil {
		return nil, time.Now(), err
	}

	zfs, err := NewZipFileSystem(zr)
	if err != nil {
		return nil, time.Now(), err
	}

	jc = new(JavadocCached)
	jc.server = zfs
	jc.size = int64(len(data))
	jc.cached = time.Now()
	jc.artifact = artifact

	h.versionCacheLock.Lock()
	h.versionCache[c] = jc
	h.tidyVersionCache()
	h.versionCacheLock.Unlock()

	return jc.server, h.calculateValidUntil(c, jc.cached), nil
}

func (h *JavadocHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pth := strings.TrimPrefix(r.URL.Path, "/")
	pieces := strings.SplitN(pth, "/", 2)

	if x, ok := h.compat[pieces[0]]; r.URL.Path == "/" || (x && ok) {
		h.versionsLock.RLock()
		var vr maven.Coordinate
		for n := len(h.versions) - 1; n >= 0; n-- {
			vr = h.versions[n]
			if excl, ok := h.excludeVersions[vr.Version]; !vr.IsSnapshot() && !(ok && excl) {
				break
			}
		}
		h.versionsLock.RUnlock()
		q := ""
		if r.URL.RawQuery != "" {
			q = "?" + r.URL.RawQuery
		}
		w.Header().Add("Location", "/"+vr.Version+r.URL.Path+q)
		w.WriteHeader(http.StatusFound)
		return
	}

	// otherwise, try and find that version
	var vr *maven.Coordinate
	h.versionsLock.Lock()
	for _, r := range h.versions {
		if r.Version == pieces[0] {
			vr = &r
			break
		}
	}
	h.versionsLock.Unlock()

	if vr == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	zf, validUntil, err := h.fetchForCoordinate(*vr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// set the cache expiry (for Fastly)
	validUntilSecondsFromNow := int64(validUntil.Sub(time.Now()).Seconds())
	browserValidUntilSecondsFromNow := validUntilSecondsFromNow
	if browserValidUntilSecondsFromNow > 600 {
		// cap browser validity to 10 minutes
		browserValidUntilSecondsFromNow = 600
	}
	w.Header().Set("Surrogate-Control", fmt.Sprintf("max-age=%d", validUntilSecondsFromNow))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", browserValidUntilSecondsFromNow))

	zfh := http.FileServer(zf)

	r.URL.Path = "/" + pieces[1]
	zfh.ServeHTTP(w, r)
	return
}

func (jh *JavadocHandler) AddCompatFor(thing string) {
	jh.compat[thing] = true
}

func (jh *JavadocHandler) populateVersions() error {
	versions, err := jh.repository.VersionsForCoordinate(jh.coordinate)
	if err != nil {
		return err
	}

	inVersions := make([]maven.Coordinate, len(versions))

	for n, v := range versions {
		v.Classifier = "javadoc"
		v.Packaging = "jar"
		inVersions[n] = v
	}

	jh.versionsLock.Lock()
	defer jh.versionsLock.Unlock()
	jh.versions = inVersions

	return nil
}

func NewJavadocHandler(repository maven.Repository, coordinate maven.Coordinate) (*JavadocHandler, error) {
	jh := new(JavadocHandler)
	jh.repository = repository
	jh.coordinate = coordinate
	jh.excludeVersions = make(map[string]bool)
	jh.versionCache = make(JavadocCache)
	jh.compat = make(map[string]bool)
	if err := jh.populateVersions(); err != nil {
		return nil, err
	}
	go jh.refresher()
	return jh, nil
}
