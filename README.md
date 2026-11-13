# TechGuild_BackEnd

## Profile country and city input

Individual profile creation and updates accept `country` and `city` as text fields.
For searchable options, the authenticated location endpoints use GeoNames:

- `GET /v1/locations/countries` returns country names and ISO alpha-2 codes.
- `GET /v1/locations/cities?country=IN&q=mun&limit=10` returns city suggestions
  for the selected country. The `q` prefix must contain at least two characters;
  `limit` is optional and capped at 10.

Set `GEONAMES_USERNAME` in the backend environment before using these endpoints.
The frontend should debounce city searches and send the selected country's
alpha-2 code. When saving an individual profile, send the selected country name
and city name in its existing `country` and `city` fields.