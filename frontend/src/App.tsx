import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import Lenis from 'lenis';
import { Search, Loader2, Sparkles, PlayCircle, Globe } from 'lucide-react';
import { AnimatePresence, motion } from 'framer-motion';

import { searchSchema, type SearchFormData } from './types/schema.ts';
import { MovieCard } from './components/MovieCard';
import { Footer } from './components/Footer';
import { fetchMovies } from './api';

export default function App() {
    const [isScrolled, setIsScrolled] = useState(false);
    const [activeQuery, setActiveQuery] = useState('');

    useEffect(() => {
        const lenis = new Lenis();
        const raf = (time: number) => {
            lenis.raf(time);
            requestAnimationFrame(raf);
        };
        requestAnimationFrame(raf);

        const handleScroll = () => setIsScrolled(window.scrollY > 20);
        window.addEventListener('scroll', handleScroll);

        return () => {
            window.removeEventListener('scroll', handleScroll);
            lenis.destroy();
        };
    }, []);

    const { register, handleSubmit, formState: { errors } } = useForm<SearchFormData>({
        resolver: zodResolver(searchSchema),
    });

    const { data: results, isFetching, isFetched } = useQuery({
        queryKey: ['movies', activeQuery],
        queryFn: () => fetchMovies(activeQuery),
        enabled: activeQuery.length > 0,
        staleTime: 1000 * 60 * 5,
    });

    const onSubmit = (data: SearchFormData) => setActiveQuery(data.query);

    return (
        <div className="min-h-screen bg-black text-white selection:bg-blue-500/40 font-sans antialiased overflow-x-hidden flex flex-col">
            <div className="fixed inset-0 overflow-hidden pointer-events-none z-0">
                <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/10 blur-[120px] animate-pulse" />
                <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px]" />
            </div>

            <nav className={`fixed top-0 w-full z-50 transition-all duration-500 border-b ${isScrolled ? 'bg-black/60 backdrop-blur-2xl border-white/10 py-3' : 'bg-transparent border-transparent py-5'}`}>
                <div className="max-w-7xl mx-auto px-4 md:px-8 flex justify-between items-center">
                    <div className="text-xl md:text-2xl font-black tracking-tighter flex items-center gap-2 cursor-pointer" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
                        <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center shadow-[0_0_20px_rgba(37,99,235,0.4)]">
                            <PlayCircle size={18} fill="white" />
                        </div>
                        SEARCH<span className="text-blue-500 font-black">CINEMA</span>
                    </div>
                    <a href="https://miktatmert.dev" target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 bg-white/5 hover:bg-white/10 px-4 py-2 rounded-full border border-white/10 transition-all text-xs font-bold tracking-widest uppercase">
                        <Globe size={14} className="text-blue-500" />
                        <span className="hidden sm:inline">Portfolio</span>
                    </a>
                </div>
            </nav>

            <main className="relative pt-32 md:pt-40 px-4 md:px-6 max-w-7xl mx-auto grow w-full">
                <div className="text-center mb-16 md:mb-24">
                    <motion.div initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-400 text-[10px] md:text-xs font-bold mb-6 md:mb-8 uppercase tracking-widest">
                        <Sparkles size={14} /> Aklında kalanları anlat, AI bulsun.
                    </motion.div>
                    <motion.h1 initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="text-4xl md:text-6xl lg:text-8xl font-black tracking-tighter mb-6 md:mb-8 bg-linear-to-b from-white via-white to-white/30 bg-clip-text text-transparent leading-tight">
                        Sadece iste, <br className="hidden sm:block" /> AI bulsun.
                    </motion.h1>
                </div>

                <form onSubmit={handleSubmit(onSubmit)} className="max-w-3xl mx-auto mb-20 md:mb-32 relative group">
                    <div className="absolute -inset-1 bg-linear-to-r from-blue-600 to-indigo-600 rounded-4xl md:rounded-[2.5rem] blur-2xl opacity-10 group-focus-within:opacity-30 transition-all duration-700" />
                    <div className={`relative flex flex-col sm:flex-row items-center bg-white/3 border ${errors.query ? 'border-red-500/50' : 'border-white/10'} rounded-2xl md:rounded-4xl p-2 backdrop-blur-3xl transition-all duration-500 focus-within:border-white/20 focus-within:bg-white/6`}>
                        <div className="flex items-center w-full px-4">
                            <Search className={errors.query ? 'text-red-500' : 'text-white/20'} size={24} />
                            <input
                                {...register('query')}
                                placeholder="Film temasını anlat..."
                                className="w-full bg-transparent px-4 py-4 md:py-6 outline-none text-lg md:text-2xl placeholder:text-white/10 font-light"
                            />
                        </div>
                        <button disabled={isFetching} className="w-full sm:w-auto bg-white text-black h-12 md:h-17 px-8 md:px-10 rounded-xl md:rounded-3xl font-black text-sm md:text-lg hover:scale-[0.97] active:scale-95 transition-all disabled:opacity-50 flex items-center justify-center gap-3 mt-2 sm:mt-0">
                            {isFetching ? <Loader2 className="animate-spin" size={20} /> : 'KEŞFET'}
                        </button>
                    </div>
                    {errors.query && (
                        <motion.p initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="absolute -bottom-8 left-6 text-red-500 text-xs font-bold uppercase tracking-widest">
                            {errors.query.message}
                        </motion.p>
                    )}
                </form>

                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 md:gap-8 pb-20">
                    <AnimatePresence mode="popLayout">
                        {results && results.length > 0 ? (
                            results.map((movie, i) => <MovieCard key={movie.ID} movie={movie} index={i} />)
                        ) : (
                            isFetched && !isFetching && activeQuery && (
                                <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="col-span-full py-20 text-center">
                                    <h2 className="text-2xl font-bold mb-2">Sonuç Bulunamadı</h2>
                                    <p className="text-white/40 text-sm max-w-xs mx-auto">Aradığın kriterlere uygun bir film bulamadık.</p>
                                </motion.div>
                            )
                        )}
                    </AnimatePresence>
                </div>
            </main>

            <Footer />
        </div>
    );
}