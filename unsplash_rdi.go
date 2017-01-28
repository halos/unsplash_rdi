package main

// Unsplash - Random Desktop

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UnRdi struct {
	Folder           string
	App_id           string
	Change_wp_secs   time.Duration
	Download_wp_secs time.Duration
	Tag              string
}

func init_prog() UnRdi {

	var unrdi UnRdi

	// Seed random generator
	rand.Seed(time.Now().UTC().UnixNano())

	// Load config
	unrdi = load_config()

	// If the folder doesn't exist, create it
	if _, err := os.Stat(unrdi.Folder); os.IsNotExist(err) {
		os.MkdirAll(unrdi.Folder, 0744)
	}

	return unrdi
}

func load_config() UnRdi {

	var unrdi UnRdi

	cwd, _ := filepath.Abs(path.Dir(os.Args[0]))
	config_path := path.Join(cwd, "config.json")

	if _, err := os.Stat(config_path); !os.IsNotExist(err) {

		// Load the config
		config_rd, err := os.Open(config_path)
		if err != nil {
			fmt.Printf("Error inesperado abriendo fichero de configuración: %v", err)
			os.Exit(1)
		}
		defer config_rd.Close()

		decoder := json.NewDecoder(config_rd)

		for decoder.More() {
			err = decoder.Decode(&unrdi)
			if err != nil {
				fmt.Printf("Error al cargar la configuración desde el JSON: %v", err)
			}
		}

	} else {

		// Create standard config and save it
		unrdi = UnRdi{Change_wp_secs: 300, Download_wp_secs: 300}
		config_wd, err := os.Create(config_path)
		if err != nil {
			fmt.Printf("Error inesperado abriendo el fichero de configuración: %v\n", err)
			os.Exit(1)
		}
		defer config_wd.Close()

		to_write, err := json.Marshal(unrdi)
		if err != nil {
			fmt.Printf("Unexpected error marshalling configuration: %v\n", err)
			os.Exit(1)
		}

		_, err = config_wd.Write(to_write)
		if err != nil {
			fmt.Printf("Error inesperado escribiendo el fichero de configuración: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration file didn't exist. Created at %v\n", config_path)
		fmt.Println("Fill it in!")

		os.Exit(0)

	}

	return unrdi
}

func (unrdi UnRdi) list_wallpapers() []string {

	var wallpapers []string

	items, error := ioutil.ReadDir(unrdi.Folder)
	if error != nil {
		fmt.Printf("Error listando archivos %v\n", error)
		// Hacer algo!!!
	}

	for _, item := range items {
		if !item.IsDir() {
			wp_name := filepath.Join(unrdi.Folder, item.Name())
			wallpapers = append(wallpapers, wp_name)
		}
	}

	return wallpapers

}

func (unrdi UnRdi) get_random_wallpaper() string {

	var wallpaper string
	var wallpapers []string

	wallpapers = unrdi.list_wallpapers()

	wallpaper = wallpapers[rand.Intn(len(wallpapers))]

	fmt.Printf("Setting new wallpaper: %v\n", wallpaper)

	return wallpaper

}

func set_wallpaper(wallpaper_path string) {

	wallpaper_uri := fmt.Sprintf("file://%s", wallpaper_path)
	command := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", wallpaper_uri)
	err := command.Run()
	if err != nil {
		fmt.Printf("Error estableciendo fondo de pantalla: %v", err)
	}

}

func (unrdi UnRdi) save_wallpaper(buff []byte, name string) {

	wallpaper_path := path.Join(unrdi.Folder, name)

	wallpaper_fd, err := os.Create(wallpaper_path)
	if err != nil {
		fmt.Printf("Error inesperado creando fichero %v: %v", name, err)
		return
	}
	defer wallpaper_fd.Close()

	_, err = wallpaper_fd.Write(buff)
	if err != nil {
		fmt.Printf("Error inesperado escribiendo en el archivo %v: %v", name, err)
	}

}

func (unrdi UnRdi) download_wallpaper(url string) {

	var name string
	fmt.Printf("Descargando fondo %v", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error inesperado descargando el fondo: %v", err)
		return
	}

	defer resp.Body.Close()

	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error inesperado leyendo la imagen: %v", err)
		return
	}

	if strings.Contains(url, "&dl=") {
		name = strings.Split(strings.Split(url, "&dl=")[1], "&")[0]
	} else {
		parts := strings.Split(url, "/")
		name = fmt.Sprintf("%v.jpg", parts[len(parts)-1])
	}

	fmt.Println("Guardándolo en ", name)

	unrdi.save_wallpaper(buff, name)

}

type images_response struct {
	//Instagram_username, Email, Id, Username, First_name, Last_name, Portfolio_url, Bio, Location string
	//Downloads, Uploads_remaining, Total_likes, Total_photos, Total_collections                   int
	//Followed_by_user                                                                             bool
	//Links                                                                                        map[string]string
	//Name, Text string
	Urls map[string]string
}

//func (unrdi UnRdi) download_random_wallpaper() []string {
func (unrdi UnRdi) download_random_wallpaper() {

	// Get JSON
	var count int = 1

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.unsplash.com/photos/random", nil)

	req.Header.Add("Accept-Version", "v1")
	req.Header.Add("Authorization", fmt.Sprintf("Client-ID %v", unrdi.App_id))

	get_params := req.URL.Query()

	get_params.Add("query", unrdi.Tag)
	get_params.Add("count", strconv.Itoa(count))
	req.URL.RawQuery = get_params.Encode()
	fmt.Println(req.URL.String())
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		fmt.Printf("[-] Error inesperado descargando el fondo: %v", err)
		fmt.Printf("[-] Error HTTP: %v\n", resp.StatusCode)
		fmt.Printf("[-] Cabeceras devueltas: %v\n", resp.Header)
		return
	}

	//fmt.Printf("[+] Remaining requests: %v / %v\n", resp.Header["X-Ratelimit-Remaining"][0], resp.Header["X-Ratelimit-Limit"][0])

	decoder := json.NewDecoder(resp.Body)

	var json_resp images_response

	_, err = decoder.Token()
	if err != nil {
		fmt.Printf("Error al leer el token: %v", err)
	}

	// Iterar la lista de elementos
	for decoder.More() {

		err = decoder.Decode(&json_resp)
		if err != nil {
			fmt.Printf("Error al decodificar: %v", err)
		}

		// Get WP URL
		raw_url := json_resp.Urls["raw"]
		fmt.Printf("%T: %v\n", raw_url, raw_url)
		go unrdi.download_wallpaper(raw_url)
	}

	_, err = decoder.Token()
	if err != nil {
		fmt.Printf("Error al leer el token: %v", err)
	}

}

func (unrdi UnRdi) set_random_wallpaper() {

	// get wp
	new_wallpaper := unrdi.get_random_wallpaper()

	// set wp
	set_wallpaper(new_wallpaper)

}

func main() {

	unrdi := init_prog()

	download_tick := time.Tick(unrdi.Download_wp_secs * time.Second)
	wp_change_tick := time.Tick(unrdi.Change_wp_secs * time.Second)

	for {
		select {
		case <-download_tick:
			unrdi.download_random_wallpaper()

		case <-wp_change_tick:
			unrdi.set_random_wallpaper()

		}

	}

}
