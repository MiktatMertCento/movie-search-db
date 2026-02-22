import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { QueryClientProvider, QueryClient } from "@tanstack/react-query";
import { GoogleReCaptchaProvider } from 'react-google-recaptcha-v3';

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            retry: 1,
            refetchOnWindowFocus: false,
            staleTime: 5 * 60 * 1000,
        },
    },
});


createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <QueryClientProvider client={queryClient}>
            <GoogleReCaptchaProvider reCaptchaKey="6LcxQHQsAAAAANytC7z-VY_UWRHm2TvFxwJXXVgS">
                <App />
            </GoogleReCaptchaProvider>
        </QueryClientProvider>
    </StrictMode>,
)
