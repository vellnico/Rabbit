import { mount } from 'svelte'
import '../src/styles/base.css'
import Setup from './Setup.svelte'

export default mount(Setup, { target: document.getElementById('app') })
