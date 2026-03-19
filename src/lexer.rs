use crate::options::ParseConfig;

const EOF: u8 = 0;

#[derive(Debug, Clone)]
pub struct LexerState {
    start: usize,
    pos: usize,
    width: bool,
    line: usize,
    num_values: usize,
}

pub struct Lexer {
    pub input: Vec<u8>,
    pub start: usize,
    pub pos: usize,
    /// true if the last next_byte actually advanced; false after EOF or backup.
    width: bool,
    pub line: usize,
    pub num_values: usize,
    pub config: ParseConfig,
}

impl Lexer {
    pub fn new(input: &[u8], config: ParseConfig) -> Self {
        Lexer {
            input: input.to_vec(),
            start: 0,
            pos: 0,
            width: false,
            line: 1,
            num_values: 0,
            config,
        }
    }

    pub fn save(&self) -> LexerState {
        LexerState {
            start: self.start,
            pos: self.pos,
            width: self.width,
            line: self.line,
            num_values: self.num_values,
        }
    }

    pub fn restore(&mut self, state: LexerState) {
        self.start = state.start;
        self.pos = state.pos;
        self.width = state.width;
        self.line = state.line;
        self.num_values = state.num_values;
    }

    pub fn next_byte(&mut self) -> u8 {
        if self.pos >= self.input.len() {
            self.width = false;
            return EOF;
        }
        let b = self.input[self.pos];
        self.pos += 1;
        self.width = true;
        if b == b'\n' {
            self.line += 1;
        }
        b
    }

    pub fn peek(&self) -> u8 {
        if self.pos >= self.input.len() {
            EOF
        } else {
            self.input[self.pos]
        }
    }

    pub fn backup(&mut self) {
        if !self.width || self.pos == 0 {
            return;
        }
        self.pos -= 1;
        self.width = true;
        if self.input[self.pos] == b'\n' {
            self.line -= 1;
        }
    }

    pub fn ignore(&mut self) {
        self.start = self.pos;
    }

    /// Take the current span as a String. Each byte is mapped to its Latin-1
    /// code point, giving a 1:1 byte-to-char mapping.
    pub fn take(&mut self) -> String {
        let val: String = self.input[self.start..self.pos]
            .iter()
            .map(|&b| b as char)
            .collect();
        self.start = self.pos;
        val
    }

    /// Take the current span as raw bytes.
    pub fn take_bytes(&mut self) -> Vec<u8> {
        let val = self.input[self.start..self.pos].to_vec();
        self.start = self.pos;
        val
    }

    pub fn accept(&mut self, valid: &[u8]) -> bool {
        let b = self.next_byte();
        if b != EOF && valid.contains(&b) {
            return true;
        }
        self.backup();
        false
    }

    pub fn accept_until(&mut self, target: u8) {
        loop {
            let b = self.next_byte();
            if b == target || b == EOF {
                if b != EOF {
                    self.backup();
                }
                return;
            }
        }
    }

    pub fn accept_run(&mut self, valid: &[u8]) {
        loop {
            let b = self.next_byte();
            if b == EOF || !valid.contains(&b) {
                self.backup();
                return;
            }
        }
    }

    pub fn skip_space_bytes(&mut self) {
        while is_space(self.next_byte()) {}
        self.backup();
    }

    /// col returns 1-based column offset within the current line.
    pub fn col(&self) -> usize {
        let mut line_start = 0;
        for i in (0..self.pos).rev() {
            if self.input[i] == b'\n' {
                line_start = i + 1;
                break;
            }
        }
        self.pos - line_start + 1
    }

    pub fn peek_string(&self) -> String {
        let end = (self.start + 10).min(self.input.len());
        self.input[self.start..end]
            .iter()
            .map(|&b| b as char)
            .collect()
    }

    /// Skip whitespace and comments, then ignore buffered content.
    pub fn skip_white_space(&mut self) -> Result<(), String> {
        self.accept_whitespace()?;
        self.ignore();
        Ok(())
    }

    /// Accept whitespace and comments (recursive for line comments).
    pub fn accept_whitespace(&mut self) -> Result<(), String> {
        self.skip_space_bytes();

        if self.peek() == b'-' {
            self.next_byte();
            if self.peek() != b'-' {
                self.backup();
            } else {
                self.next_byte(); // consume second '-'

                // Check for block comment: --[[ ... ]]
                let mut skipped_block_comment = false;
                if self.peek() == b'[' {
                    self.next_byte(); // consume '['
                    let mut pattern = vec![b']'];
                    while self.next_byte() == b'#' {
                        pattern.push(b'#');
                    }
                    self.backup();

                    if self.peek() == b'[' {
                        // Block comment
                        pattern.push(b']');
                        let found = self.accept_through_pattern(&pattern);
                        if found {
                            skipped_block_comment = true;
                        } else {
                            let pat_str: String = pattern.iter().map(|&b| b as char).collect();
                            return Err(format!(
                                "multiline string not properly closed, looking for {:?}",
                                pat_str
                            ));
                        }
                    } else {
                        // Not a block comment, back up the non-'[' char
                        self.backup();
                    }
                } else {
                    self.backup();
                }

                if !skipped_block_comment {
                    // Line comment: skip until newline
                    self.accept_until(b'\n');
                    self.accept_whitespace()?;
                }
            }
        }

        Ok(())
    }

    fn accept_through_pattern(&mut self, pattern: &[u8]) -> bool {
        if pattern.is_empty() {
            return true;
        }

        let first_byte = pattern[0];

        self.accept_until(first_byte);

        let mut found_mismatch = false;
        for &pb in pattern {
            let nb = self.next_byte();
            if nb == EOF {
                found_mismatch = true;
                break;
            }
            if nb != pb {
                found_mismatch = true;
                break;
            }
        }

        !found_mismatch
    }
}

fn is_space(b: u8) -> bool {
    b == b' ' || b == b'\t' || b == b'\r' || b == b'\n'
}

pub fn is_alpha_numeric(b: u8) -> bool {
    b == b'_' || b.is_ascii_alphabetic() || b.is_ascii_digit()
}

/// Decode raw bytes into a String. If the bytes are valid UTF-8, decode as
/// UTF-8 (so "Fröst" renders correctly). Otherwise, map each byte to its
/// Latin-1 code point (so binary blobs are preserved losslessly).
pub fn bytes_to_string(bytes: &[u8]) -> String {
    match std::str::from_utf8(bytes) {
        Ok(s) => s.to_string(),
        Err(_) => bytes.iter().map(|&b| b as char).collect(),
    }
}
