# Location

Export entities from Home Assistant for the purposes of presenting on a map

## Building

### Simple

```bash
cd cmd/location
go build
```

### Docker (say for a raspberry pi 4)

```bash
docker buildx build --platform linux/arm64 -t registry/path/location .
docker push registry/path/location
```

## Arguments

- `-concurrency` Polling concurrency (default 2)
- `-entity` Entity ID to export, repeat flag or comma separate for more
- `-listen` Listen configuration for HTTP traffic (default ":9922")
- `-log-level` Log level(default "info")
- `-poll-interval` Rate of polling [duration](#duration) (default 5s)
- `-ring` Switches to using a ring buffer of the given size for historical storage
- `-bootstrap` Ask Home Assistant for event history on startup for the given [duration](#duration)
- `-filter` Filter out some useless/bad data (missing coordinates, no position lock)
- `-token` Home Assistant token [env: HA_TOKEN]
- `-url` Home Assistant URL [env: HA_URL]

## Environment

As you may have noticed, you can opt to provide `HA_TOKEN` and `HA_URL` instead of using the `-token` and -url `flags`

## Duration

A duration is a sequence of decimal numbers, each with optional fraction and a unit suffix
such as "300ms", "1.5h" or "2h45m".

Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

## Use

You can build yourself a map in a html page, I used leafelet with openstreetmap tiles, and set up an EventSource

For example.

```javascript
    var states = {}

	const bikeIcon = L.IconMaterial.icon({
	  	icon: 'motorcycle',                // Name of Material icon
		iconColor: '#aa2187',              // Material icon color (could be rgba, hex, html name...)
		markerColor: 'rgba(255,0,0,0.5)',  // Marker fill color
		outlineColor: 'yellow',            // Marker outline color
		outlineWidth: 1,                   // Marker outline width
		iconSize: [31, 42]                 // Width and height of the icon
  	})

	var bike = L.marker([-20.7258195,139.4884129], {icon: bikeIcon, title:"bike", alt:"bike"}).addTo(map);

	const evtSource = new EventSource("http://localhost:9922/sse")
	evtSource.onmessage = (event) => {
		var obj = JSON.parse(event.data)
		var pos = [obj.attributes.latitude, obj.attributes.longitude]

		if (obj.entity_id.match(/bike/)) {
			bike.setLatLng(pos)
			states["bike"] = obj
		}
	}
```