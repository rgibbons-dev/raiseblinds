import { createMemo, createSignal, For, Show } from 'solid-js'
import { filterListings, sampleListings } from './market'

export default function App() {
  const [query, setQuery] = createSignal('')
  const [selected, setSelected] = createSignal(sampleListings[0])
  const [vouch, setVouch] = createSignal(true)
  const [rating, setRating] = createSignal(5)
  const items = createMemo(() => filterListings(sampleListings, query()))

  return (
    <main style={{ 'font-family': 'Inter, ui-sans-serif', padding: '1rem', 'max-width': '1100px', margin: '0 auto', color: '#111827' }}>
      <h1 style={{ color: '#B91C1C', 'font-size': '2rem', 'font-weight': 800 }}>RaiseBlinds Marketplace</h1>
      <p style={{ color: '#374151' }}>Local buy/sell app with reviews, vouches, and map view.</p>
      <section style={{ display: 'grid', gap: '0.75rem', 'grid-template-columns': '2fr 1fr' }}>
        <input value={query()} onInput={(e) => setQuery(e.currentTarget.value)} placeholder="Search listed items" style={{ padding: '0.75rem', border: '1px solid #E5E7EB', 'border-radius': '12px' }} />
        <button style={{ background: '#B91C1C', color: 'white', border: 'none', 'border-radius': '12px', 'font-weight': 700 }}>Login / Sign up</button>
      </section>
      <section style={{ display: 'grid', 'grid-template-columns': '1.4fr 1fr', gap: '1rem', 'margin-top': '1rem' }}>
        <div style={{ display: 'grid', gap: '0.75rem' }}>
          <For each={items()}>{(item) => (
            <article onClick={() => setSelected(item)} style={{ border: '1px solid #E5E7EB', 'border-radius': '16px', padding: '0.75rem', cursor: 'pointer', 'box-shadow': '0 1px 2px rgba(0,0,0,0.05)' }}>
              <img src={item.imageUrl} alt={item.title} style={{ width: '100%', height: '180px', 'object-fit': 'cover', 'border-radius': '12px' }} />
              <h3>{item.title}</h3>
              <p>{item.description}</p>
              <strong>${(item.priceCents / 100).toFixed(2)}</strong>
            </article>
          )}</For>
        </div>
        <aside style={{ border: '1px solid #E5E7EB', 'border-radius': '16px', padding: '0.75rem' }}>
          <h3 style={{ color: '#B91C1C' }}>Map View</h3>
          <Show when={selected()}>
            {(s) => (
              <iframe title="map" style={{ width: '100%', height: '220px', border: 0 }} src={`https://www.openstreetmap.org/export/embed.html?bbox=${s().lng - 0.01}%2C${s().lat - 0.01}%2C${s().lng + 0.01}%2C${s().lat + 0.01}&layer=mapnik&marker=${s().lat}%2C${s().lng}`}></iframe>
            )}
          </Show>
          <h4>Leave a review / vouch</h4>
          <label><input type="checkbox" checked={vouch()} onInput={(e) => setVouch(e.currentTarget.checked)} /> Vouch for seller</label>
          <label style={{ display: 'block', 'margin-top': '0.5rem' }}>Rating
            <input type="number" min="1" max="5" value={rating()} onInput={(e) => setRating(parseInt(e.currentTarget.value || '5'))} />
          </label>
          <p style={{ 'font-size': '0.9rem', color: '#6B7280' }}>Sample mode uses stock images and seeded demo data.</p>
        </aside>
      </section>
    </main>
  )
}
