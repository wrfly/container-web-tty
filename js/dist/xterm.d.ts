/// <reference types="node" />
import { lib } from "libapps";
import { FitAddon } from 'xterm-addon-fit';
import { Terminal } from 'xterm';
export declare class Xterm {
    elem: HTMLElement;
    term: Terminal;
    fitAddon: FitAddon;
    resizeListener: () => void;
    decoder: lib.UTF8Decoder;
    message: HTMLElement;
    messageTimeout: number;
    messageTimer: NodeJS.Timeout;
    constructor(elem: HTMLElement);
    info(): {
        columns: number;
        rows: number;
    };
    output(data: string): void;
    showMessage(message: string, timeout: number): void;
    removeMessage(): void;
    setWindowTitle(title: string): void;
    setPreferences(value: object): void;
    onInput(callback: (input: string) => void): void;
    onResize(callback: (colmuns: number, rows: number) => void): void;
    deactivate(): void;
    reset(): void;
    close(): void;
}
