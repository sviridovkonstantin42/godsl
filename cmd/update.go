package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"github.com/sviridovkonstantin42/godsl/internal/revision"
	"github.com/sviridovkonstantin42/godsl/internal/utils/consts"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Обновляет godsl",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()

		newVersion, err := update(ctx)
		if err != nil {
			fmt.Printf("Ошибка при обновлении: %v\n", err)
			return
		}

		if newVersion != "" {
			fmt.Printf("Обновился с %v на %v!\nРекомендую перегенерировать код командой 'godsl generate' \n",
				revision.Revision, newVersion)

		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func update(ctx context.Context) (newVersion string, err error) {
	client := github.NewClient(nil)
	latestTag, err := getLatestTag(ctx, client, consts.OWNER, consts.REPOSITORY)
	if err != nil {
		return "", err
	}

	if latestTag == revision.Revision {
		fmt.Println("Используется самая свежая версия.")
		return "", nil
	}

	platform, err := getPlatform()
	if err != nil {
		return "", err
	}

	archType, err := getArchType()
	if err != nil {
		return "", err
	}

	downloadedLatestRelease, err := downloadLatestRealese(ctx, client, platform, archType)
	if err != nil {
		return "", err
	}

	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	return newVersion, os.Rename(downloadedLatestRelease, executable)
}

func getLatestTag(ctx context.Context, client *github.Client, owner, repo string) (string, error) {
	tags, _, err := client.Repositories.ListTags(ctx, owner, repo, nil)
	if err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("не найдены теги")
	}

	return tags[0].GetName(), nil
}

func getPlatform() (string, error) {
	os := runtime.GOOS
	switch os {
	case "linux":
		return "linux", nil
	case "darwin":
		return "macOS", nil
	default:
		return "", fmt.Errorf("неподддерживаемая OS: %s", os)
	}
}

func getArchType() (string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64", nil
	case "arm64", "arm":
		return "arm64", nil
	default:
		return "", fmt.Errorf("неподдерживаемая архитектура: %s", arch)
	}
}

func downloadLatestRealese(ctx context.Context, client *github.Client, platform, archType string) (string, error) {
	releases, _, err := client.Repositories.ListReleases(ctx, consts.OWNER, consts.REPOSITORY, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка при получении списка релизов: %v", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("нет доступных релизов")
	}

	latestRelease := releases[0]
	if len(latestRelease.Assets) == 0 {
		return "", fmt.Errorf("у релиза нет бинарных файлов")
	}

	var asset *github.ReleaseAsset

	for _, a := range latestRelease.Assets {
		if a.Name == nil {
			continue
		}

		if matchesTarget(*a.Name, platform, archType) {
			asset = &a
			break
		}
	}

	if asset == nil {
		return "", fmt.Errorf("не найден подходящий файл для вашей платформы")
	}

	downloadUrl := asset.BrowserDownloadURL

	tempDir := os.TempDir()
	zipFile := filepath.Join(tempDir, *asset.Name)
	out, err := os.Create(zipFile)
	if err != nil {
		return "", fmt.Errorf("не удалось создать временный файл: %v", err)
	}
	defer out.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", *downloadUrl, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при скачивании файла: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка HTTP: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при сохранении файла: %v", err)
	}

	err = exec.Command("tar", "--no-same-owner", "-xzf", zipFile, "-C", tempDir).Run()
	if err != nil {
		return "", err
	}

	execFile := filepath.Join(tempDir, "godsl")

	if runtime.GOOS != "windows" {
		err = os.Chmod(execFile, 0755)
		if err != nil {
			return "", fmt.Errorf("не удалось установить права на выполнение: %v", err)
		}
	}

	return execFile, nil
}

func matchesTarget(filename, targetOS, targetArch string) bool {
	return strings.Contains(filename, targetOS) && strings.Contains(filename, targetArch)
}
