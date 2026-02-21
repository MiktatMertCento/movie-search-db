import { Github, Globe } from 'lucide-react';

export const Footer = () => {
    const currentYear = new Date().getFullYear();

    return (
        <footer className="relative z-10 border-t border-white/10 bg-black/50 backdrop-blur-md py-12 px-6">
            <div className="max-w-7xl mx-auto flex flex-col md:flex-row justify-between items-center gap-8">
                <div className="text-center md:text-left">
                    <div className="text-xl font-black tracking-tighter mb-2 uppercase">
                        SEARCH<span className="text-blue-500">CINEMA</span>
                    </div>
                    <p className="text-white/40 text-xs font-medium max-w-xs leading-relaxed">
                        Yapay zeka ve vektör veritabanı teknolojileri kullanılarak geliştirilmiş
                        anlamsal film arama motoru.
                    </p>
                </div>

                <div className="flex flex-col items-center md:items-end gap-4">
                    <div className="flex gap-4">
                        <a
                            href="https://github.com/MiktatMertCento"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-3 rounded-full bg-white/5 border border-white/10 hover:border-blue-500/50 hover:text-blue-500 transition-all duration-300"
                        >
                            <Github size={20} />
                        </a>
                        <a
                            href="https://miktatmert.dev"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-3 rounded-full bg-white/5 border border-white/10 hover:border-blue-500/50 hover:text-blue-500 transition-all duration-300"
                        >
                            <Globe size={20} />
                        </a>
                    </div>
                    <p className="text-[10px] font-bold tracking-[0.3em] uppercase text-white/20">
                        &copy; {currentYear} <a href="https://miktatmert.dev" className="hover:text-blue-500 transition-colors">Miktat Mert Cento</a>. Tüm Hakları Saklıdır.
                    </p>
                </div>
            </div>
        </footer>
    );
};