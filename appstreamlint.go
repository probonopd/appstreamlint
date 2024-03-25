package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type Launchable struct {
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

type Component struct {
	XMLName         xml.Name     `xml:"component"`
	Type            string       `xml:"type,attr"`
	ID              string       `xml:"id"`
	Name            string       `xml:"name"`
	Summary         string       `xml:"summary"`
	MetadataLicense string       `xml:"metadata_license"`
	ProjectLicense  string       `xml:"project_license"`
	Description     string       `xml:"description"`
	Launchable      Launchable   `xml:"launchable"`
	Screenshots     []Screenshot `xml:"screenshots>screenshot"`
}

type Screenshot struct {
	Caption     string `xml:"caption"`
	Image       Image  `xml:"image"`
	Environment string `xml:"environment,attr"`
}

type Image struct {
	Type   string `xml:"type,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Source string `xml:",chardata"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: appstreamlint <file>")
		os.Exit(1)
	}
	filePath := os.Args[1]
	xmlFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	var component Component
	if err := xml.Unmarshal(byteValue, &component); err != nil {
		fmt.Println("Error parsing XML:", err)
		os.Exit(1)
	}

	// Check filename
	// While desktop-application metadata is commonly stored in /usr/share/metainfo/%{id}.metainfo.xml (with a .metainfo.xml extension),
	// using a .appdata.xml extension is also permitted for this component type for legacy compatibility.
	// NOTE: This implementation will accept both .metainfo.xml and .appdata.xml extensions because the AppStream format should always be backwards compatible.
	if filePath != component.ID+".appdata.xml" && filePath != component.ID+".metainfo.xml" {
		fmt.Println("Error: Filename must be the same as the ID with .appdata.xml extension")
		// Print the correct filename
		fmt.Println("Correct filename:", component.ID+".metainfo.xml")
		// Print the actual filename
		fmt.Println("Actual filename:", filePath)
		os.Exit(1)
	}

	// Check required fields
	requiredFields := map[string]string{
		"Type":            component.Type,
		"ID":              component.ID,
		"Name":            component.Name,
		"Summary":         component.Summary,
		"MetadataLicense": component.MetadataLicense,
		"ProjectLicense":  component.ProjectLicense,
		"Description":     component.Description,
		"LaunchableType":  component.Launchable.Type,
		"Launchable":      component.Launchable.Contents,
	}

	for field, value := range requiredFields {
		if value == "" {
			fmt.Printf("Error: %s must not be empty\n", field)
			os.Exit(1)
		}
	}

	// The desktop-application component type is the same as the desktop component type -
	// desktop is the older type identifier for desktop-applications and should not be used for new metainfo files,
	// unless compatibility with very old AppStream tools (pre 2016) is still wanted.
	// NOTE: Both types will be accepted in this implementation, because the AppStream format should always be backwards compatible.
	if component.Type != "desktop-application" && component.Type != "desktop" {
		fmt.Println("Error: Type must be 'desktop-application'")
		os.Exit(1)
	}

	// For desktop applications, the <id/> tag value must follow the reverse-DNS scheme
	// (e.g. org.gnome.gedit, org.kde.dolphin, etc.) and must not contain any spaces or special characters.
	// NOTE: Reverse-DNS is not enforced in this implementation as we think it complicates things, especially for new developers without a domain.
	// Furthermore, the <id/> tag value used to contain the name of the desktop file with the .desktop extension, unforunately, this has changed over time
	// but the <id/> tag value should not change once it has been set for a given application.

	// https://www.freedesktop.org/software/appstream/docs/chap-Metadata.html#tag-metadata_license
	// NOTE: The AppStream specification might allow more licenses than the ones listed below in the future. So this implementation will only inform the user
	// if the metadata license is not allowed. The user can then check the latest AppStream specification for the allowed licenses.
	allowedLicenses := []string{
		"FSFAP",
		"MIT",
		"0BSD",
		"CC0-1.0",
		"CC-BY-3.0",
		"CC-BY-4.0",
		"CC-BY-SA-3.0",
		"CC-BY-SA-4.0",
		"GFDL-1.1",
		"GFDL-1.2",
		"GFDL-1.3",
		"BSL-1.0",
		"FTL",
		"FSFUL",
	}

	allowed := false
	for _, license := range allowedLicenses {
		if component.MetadataLicense == license {
			allowed = true
			break
		}
	}
	if !allowed {
		fmt.Println("Warning: Metadata license is not allowed")
		fmt.Println("Allowed licenses:", allowedLicenses)
		fmt.Println("Actual license:", component.MetadataLicense)
	}

	// The human-readable name of the application. This is the name you want users to see prior to installing the application.
	// Check that it is at least 2 characters long.
	if len(component.Name) < 2 {
		fmt.Println("Error: Name must be at least 2 characters long")
		os.Exit(1)
	}

	// A short summary on what this application does, roughly equivalent to the Comment field of the accompanying .desktop file of the application.
	// Check that it is at least 10 characters long.
	if len(component.Summary) < 10 {
		fmt.Println("Error: Summary must be at least 10 characters long")
		os.Exit(1)
	}

	// The <launchable/> tag has a required type property indicating the system that is used to launch the component. The following types are allowed:
	// desktop-id: The component is launched using a desktop file. The desktop file is identified by the <id/> tag value.
	// NOTE: This implementation will only accept the desktop-id type.
	// <launchable type="desktop-id">myapplication.desktop</launchable>
	// So check the type attribute and the value.
	if component.Launchable.Type != "desktop-id" {
		fmt.Println("Error: Launchable type must be 'desktop-id'")
		fmt.Println("Actual launchable type:", component.Launchable.Type)
		os.Exit(1)
	}

	// https://www.freedesktop.org/software/appstream/docs/chap-Metadata.html#tag-screenshots

	if len(component.Screenshots) == 0 {
		fmt.Println("Warning: No screenshots tag found")
	} else {
		if len(component.Screenshots) == 0 {
			fmt.Println("Error: No screenshot tag found inside screenshots tag")
			os.Exit(1)
		}

		for _, screenshot := range component.Screenshots {
			if screenshot.Image.Type != "source" && screenshot.Image.Type != "video" && screenshot.Image.Type != "" {
				fmt.Println("Error: Image type must be 'source' or 'video'")
				os.Exit(1)
			}
			if screenshot.Image.Type == "source" {

				// The image source must be a valid URL, starting with http:// or https:// and following RFC 3986.
				// NOTE: For simplicity, this implementation will only check if the source starts with http:// or https:// and ends with a valid image extension.
				if len(screenshot.Image.Source) < 7 || (screenshot.Image.Source[:7] != "http://" && screenshot.Image.Source[:8] != "https://") {
					fmt.Println("Error: Image source must start with http:// or https://")
					os.Exit(1)
				}
				// Check if the source ends with a valid image extension
				validExtensions := []string{".png", ".jpg", ".jpeg"} // NOTE: It is debatable whether other image extensions should be allowed
				valid := false
				for _, ext := range validExtensions {
					if len(screenshot.Image.Source) > len(ext) && screenshot.Image.Source[len(screenshot.Image.Source)-len(ext):] == ext {
						valid = true
						break
					}
				}
				if !valid {
					fmt.Println("Error: Image source must end with a valid image extension")
					fmt.Println("Valid extensions:", validExtensions)
					fmt.Println("Actual extension:", screenshot.Image.Source[len(screenshot.Image.Source)-4:])
					os.Exit(1)
				}
			}
		}

	}

	fmt.Println("Validation complete.")
}
