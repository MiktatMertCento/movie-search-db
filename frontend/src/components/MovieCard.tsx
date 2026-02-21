import {motion} from 'framer-motion';
import {Info, Star} from 'lucide-react';
import type {Movie} from '../types/schema.ts';

export const MovieCard = ({ movie, index }: { movie: Movie; index: number }) => {

    const handleClickMovieDetail = () => window.open(`https://www.themoviedb.org/movie/${movie.TmdbID}/`, '_blank');

    return (
        <motion.div
            layout
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.9 }}
            transition={{ duration: 0.5, delay: index * 0.03 }}
            className="group relative"
        >
            <div className="aspect-2/3 relative rounded-3xl md:rounded-4xl overflow-hidden bg-white/5 border border-white/5 shadow-2xl transition-all duration-500 group-hover:shadow-blue-500/10 group-hover:border-white/20">
                <img
                    src={movie.Post ? `https://image.tmdb.org/t/p/w500${movie.Post}` : 'https://via.placeholder.com/500x750'}
                    alt={movie.Title}
                    className="object-cover w-full h-full group-hover:scale-105 transition-transform duration-700"
                />
                <div className="absolute top-4 left-4 flex gap-2">
                    <div className="bg-black/60 backdrop-blur-xl px-3 py-1.5 rounded-xl text-[10px] font-black border border-white/10 flex items-center gap-1.5">
                        <Star size={10} className="text-blue-500 fill-blue-500" />
                        {movie.Vote?.toFixed(1) || "0.0"}
                    </div>
                </div>
                <div className="absolute top-4 right-4">
                    <div className="bg-blue-600 px-3 py-1.5 rounded-xl text-[10px] font-black text-white shadow-lg">
                        %{Math.round(movie.Sim * 100)} MATCH
                    </div>
                </div>
                <div className="absolute inset-0 bg-linear-to-t from-black via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
                <div className="absolute bottom-0 left-0 right-0 p-6 translate-y-10 group-hover:translate-y-0 opacity-0 group-hover:opacity-100 transition-all duration-500">
                    <p className="text-white/80 text-[10px] md:text-xs line-clamp-4 leading-relaxed mb-4 font-medium uppercase tracking-tight">
                        {movie.Ov}
                    </p>
                    <button onClick={handleClickMovieDetail} className="w-full bg-white text-black py-3 rounded-xl font-black text-[10px] tracking-widest flex items-center justify-center gap-2 hover:bg-blue-500 hover:text-white transition-colors uppercase cursor-pointer">
                        İncele <Info size={14} />
                    </button>
                </div>
            </div>
            <div className="mt-5 px-1">
                <h3 className="font-bold text-lg md:text-xl tracking-tight leading-tight group-hover:text-blue-500 transition-colors mb-1 uppercase italic line-clamp-1">{movie.Title}</h3>
                <p className="text-white/20 text-[9px] md:text-[10px] font-bold tracking-[0.2em] uppercase truncate">
                    {movie.Tag || 'Premium AI Selection'}
                </p>
            </div>
        </motion.div>
    );
};