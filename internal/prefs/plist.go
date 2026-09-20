package prefs

import (
	"encoding/xml"
	"fmt"
	"io"
)

// parseKeyEquivalents extracts the NSUserKeyEquivalents dictionary from an XML
// property list as produced by `defaults export <domain> -`.
//
// Only that one entry is decoded; every other entry is skipped without being
// interpreted. Converting the whole property list first is not an option
// because real application domains contain values with no JSON or Go
// equivalent (`plutil -convert json` rejects Chrome's domain outright).
//
// A property list without the entry yields an empty map and a nil error.
func parseKeyEquivalents(r io.Reader) (map[string]string, error) {
	dec := xml.NewDecoder(r)

	// The first <dict> encountered is the root dictionary of the plist.
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return map[string]string{}, nil
		}
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "dict" {
			return scanRootDict(dec)
		}
	}
}

func scanRootDict(dec *xml.Decoder) (map[string]string, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "key" {
				return nil, fmt.Errorf("unexpected <%s> where a <key> was expected", t.Name.Local)
			}
			var name string
			if err := dec.DecodeElement(&name, &t); err != nil {
				return nil, err
			}

			value, err := nextStartElement(dec)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", name, err)
			}
			if name != Key {
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			if value.Name.Local != "dict" {
				return nil, fmt.Errorf("%s is a <%s>, want a <dict>", Key, value.Name.Local)
			}
			return scanStringDict(dec)

		case xml.EndElement:
			// End of the root dictionary: the entry is not set.
			return map[string]string{}, nil
		}
	}
}

// scanStringDict reads the body of a <dict> whose values must all be strings.
func scanStringDict(dec *xml.Decoder) (map[string]string, error) {
	entries := map[string]string{}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "key" {
				return nil, fmt.Errorf("unexpected <%s> in %s", t.Name.Local, Key)
			}
			var menu string
			if err := dec.DecodeElement(&menu, &t); err != nil {
				return nil, err
			}

			value, err := nextStartElement(dec)
			if err != nil {
				return nil, fmt.Errorf("%s[%q]: %w", Key, menu, err)
			}
			if value.Name.Local != "string" {
				return nil, fmt.Errorf("%s[%q] is a <%s>, want a <string>", Key, menu, value.Name.Local)
			}
			var equivalent string
			if err := dec.DecodeElement(&equivalent, &value); err != nil {
				return nil, err
			}
			entries[menu] = equivalent

		case xml.EndElement:
			return entries, nil
		}
	}
}

// nextStartElement returns the next element start, skipping whitespace.
func nextStartElement(dec *xml.Decoder) (xml.StartElement, error) {
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return xml.StartElement{}, fmt.Errorf("missing value")
		}
		if err != nil {
			return xml.StartElement{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return t, nil
		case xml.EndElement:
			return xml.StartElement{}, fmt.Errorf("missing value")
		}
	}
}
