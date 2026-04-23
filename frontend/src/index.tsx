import { render } from 'solid-js/web'
import App from './App'

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => navigator.serviceWorker.register('/service-worker.js'))
}

render(() => <App />, document.getElementById('root')!)
