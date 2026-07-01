package opf

import (
	"encoding/xml"
)

// Package represents the root <package> element.
type Package struct {
	XMLName          xml.Name `xml:"package"`
	UniqueIdentifier string   `xml:"unique-identifier,attr"`
	Version          string   `xml:"version,attr"`
	Metadata         Metadata `xml:"metadata"`
	Guide            Guide    `xml:"guide"`
}

// Metadata contains all the Dublin Core and OPF metadata elements.
type Metadata struct {
	Identifiers  []Identifier `xml:"identifier"`
	Title        string       `xml:"title"`
	Creators     []Person     `xml:"creator"`
	Contributors []Person     `xml:"contributor"`
	Date         string       `xml:"date"`
	Description  string       `xml:"description"`
	Publisher    string       `xml:"publisher"`
	Language     string       `xml:"language"`
	Subjects     []string     `xml:"subject"`
	MetaElements []Meta       `xml:"meta"`
}

func (m *Metadata) Identifier(scheme string) string {
	for _, i := range m.Identifiers {
		if i.Scheme == scheme {
			return i.Value
		}
	}
	return ""
}

// Identifier maps to <dc:identifier>.
type Identifier struct {
	Value  string `xml:",chardata"`
	Scheme string `xml:"scheme,attr"` // Matches opf:scheme
	ID     string `xml:"id,attr"`
}

// Person maps to both <dc:creator> and <dc:contributor> since they share the same attribute structure.
type Person struct {
	Value  string `xml:",chardata"`
	FileAs string `xml:"file-as,attr"` // Matches opf:file-as
	Role   string `xml:"role,attr"`    // Matches opf:role
}

// Meta maps to the custom <meta> tags used by Calibre.
type Meta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// Guide represents the <guide> section.
type Guide struct {
	References []Reference `xml:"reference"`
}

// Reference maps to <reference> items inside the guide.
type Reference struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}
