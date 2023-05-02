# Unicode Property Builder

This tool extracts binary property data from the Unicode Character Database at
[unicode.org](http://unicode.org). Each property is first turned into an
internal instance of BinaryProperty. It is then emitted as a set of files
containing initialiers for Go RangeTable instances. Future versions will likely
support emission in other languages.

The intended use of this tool is to update the source trees in the adjacent
Unicode repository. By default, files are pulled directly from the unicode.org
site. For debugging convenience, files can also be downloaded locally using the
data/fetch.sh shell script.

The process of building these properties is, ahem, delightfully obscure. In
addition to extracting content from various constituent files of the UCD, some
amount of external knowledge is needed to generate composite properties such as
the merged General Categories. Potentially helpful hints may be found in several documents:

- [NamesList.html](https://www.unicode.org/Public/UCD/latest/ucd/NamesList.html)
- [UAX #38](https://www.unicode.org/reports/tr38/tr38-33.html), _Unicode Han
  Database (Unihan)_
- [UAX #42](https://www.unicode.org/reports/tr42/tr42-32.html), _Unicode
  Character Database in XML_
- [UAX #44](https://www.unicode.org/reports/tr44/tr44-30.html), _Unicode
  Character Database)_
- [UTS #51](https://unicode.org/reports/tr51/), _Unicode Emoji_
