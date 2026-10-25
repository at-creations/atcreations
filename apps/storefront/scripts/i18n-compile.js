// Generates src/lib/paraglide with the same options as the Vite plugin.
import { compile } from '@inlang/paraglide-js';
import { paraglideOptions } from '../paraglide.config.js';

await compile(paraglideOptions);
