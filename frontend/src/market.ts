export type Listing = {
  id: number
  title: string
  description: string
  priceCents: number
  lat: number
  lng: number
  imageUrl: string
}

export const sampleListings: Listing[] = [
  { id: 1, title: 'City Bike', description: 'Commuter bike in good condition', priceCents: 22000, lat: 37.7749, lng: -122.4194, imageUrl: 'https://images.unsplash.com/photo-1485965120184-e220f721d03e' },
  { id: 2, title: 'Vintage Chair', description: 'Mid-century modern chair', priceCents: 4500, lat: 37.784, lng: -122.409, imageUrl: 'https://images.unsplash.com/photo-1555041469-a586c61ea9bc' },
]

export function filterListings(items: Listing[], query: string): Listing[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  return items.filter((item) => `${item.title} ${item.description}`.toLowerCase().includes(q))
}
