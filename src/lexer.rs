use crate::options::ParseConfig;

const EOF: char = '\0';

#[derive(Debug, Clone)]
pub struct LexerState {
    start: usize,
    pos: usize,
    width: bool,
    line: usize,
    num_values: usize,
}

pub struct Lexer {
    pub input: Vec<char>,
    pub start: usize,
    pub pos: usize,
    /// true if the last next_rune actually advanced; false after EOF or backup.
    width: bool,
    pub line: usize,
    pub num_values: usize,
    pub config: ParseConfig,
}

impl Lexer {
    pub fn new(input: &str, config: ParseConfig) -> Self {
        Lexer {
            input: input.chars().collect(),
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

    pub fn next_rune(&mut self) -> char {
        if self.pos >= self.input.len() {
            self.width = false;
            return EOF;
        }
        let r = self.input[self.pos];
        self.pos += 1;
        self.width = true;
        if r == '\n' {
            self.line += 1;
        }
        r
    }

    pub fn peek(&self) -> char {
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
        self.width = true; // still valid to backup further (like Go's behavior)
        if self.input[self.pos] == '\n' {
            self.line -= 1;
        }
    }

    pub fn ignore(&mut self) {
        self.start = self.pos;
    }

    pub fn take(&mut self) -> String {
        let val: String = self.input[self.start..self.pos].iter().collect();
        self.start = self.pos;
        val
    }

    pub fn accept(&mut self, valid: &str) -> bool {
        let r = self.next_rune();
        if r != EOF && valid.contains(r) {
            return true;
        }
        self.backup();
        false
    }

    pub fn accept_until(&mut self, target: char) {
        loop {
            let r = self.next_rune();
            if r == target || r == EOF {
                if r != EOF {
                    self.backup();
                }
                return;
            }
        }
    }

    pub fn accept_run(&mut self, valid: &str) {
        loop {
            let r = self.next_rune();
            if r == EOF || !valid.contains(r) {
                self.backup();
                return;
            }
        }
    }

    pub fn skip_space_runes(&mut self) {
        while is_space(self.next_rune()) {}
        self.backup();
    }

    /// col returns 1-based column offset within the current line.
    pub fn col(&self) -> usize {
        let mut line_start = 0;
        for i in (0..self.pos).rev() {
            if self.input[i] == '\n' {
                line_start = i + 1;
                break;
            }
        }
        self.pos - line_start + 1
    }

    pub fn peek_string(&self) -> String {
        let end = (self.start + 10).min(self.input.len());
        self.input[self.start..end].iter().collect()
    }

    /// Skip whitespace and comments, then ignore buffered content.
    pub fn skip_white_space(&mut self) -> Result<(), String> {
        self.accept_whitespace()?;
        self.ignore();
        Ok(())
    }

    /// Accept whitespace and comments (recursive for line comments).
    pub fn accept_whitespace(&mut self) -> Result<(), String> {
        self.skip_space_runes();

        if self.peek() == '-' {
            self.next_rune();
            if self.peek() != '-' {
                self.backup();
            } else {
                self.next_rune(); // consume second '-'

                // Check for block comment: --[[ ... ]]
                let mut skipped_block_comment = false;
                if self.peek() == '[' {
                    self.next_rune(); // consume '['
                    let mut pattern = String::from("]");
                    while self.next_rune() == '#' {
                        pattern.push('#');
                    }
                    self.backup();

                    if self.peek() == '[' {
                        // Block comment
                        pattern.push(']');
                        let found = self.accept_through_pattern(&pattern);
                        if found {
                            skipped_block_comment = true;
                        } else {
                            return Err(format!(
                                "multiline string not properly closed, looking for {:?}",
                                pattern
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
                    self.accept_until('\n');
                    self.accept_whitespace()?;
                }
            }
        }

        Ok(())
    }

    fn accept_through_pattern(&mut self, pattern: &str) -> bool {
        if pattern.is_empty() {
            return true;
        }

        let pattern_chars: Vec<char> = pattern.chars().collect();
        let first_rune = pattern_chars[0];

        self.accept_until(first_rune);

        let mut found_mismatch = false;
        for &pc in &pattern_chars {
            let nr = self.next_rune();
            if nr == EOF {
                found_mismatch = true;
                break;
            }
            if nr != pc {
                found_mismatch = true;
                break;
            }
        }

        !found_mismatch
    }
}

fn is_space(r: char) -> bool {
    r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

pub fn is_alpha_numeric(r: char) -> bool {
    r == '_' || r.is_alphabetic() || r.is_ascii_digit()
}
