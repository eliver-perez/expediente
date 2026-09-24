"""Generate our own synthetic PDF fixtures; no customer documents or external corpus."""
from pathlib import Path
import re
import subprocess
import tempfile
import zlib

ROOT = Path(__file__).resolve().parents[1] / 'testdata/documents'
ROOT.mkdir(exist_ok=True)

def pdf(objects):
    output = bytearray(b'%PDF-1.4\n')
    offsets = [0]
    for number, value in enumerate(objects, 1):
        offsets.append(len(output))
        output += f'{number} 0 obj\n'.encode() + value + b'\nendobj\n'
    start = len(output)
    output += f'xref\n0 {len(offsets)}\n0000000000 65535 f \n'.encode()
    for offset in offsets[1:]:
        output += f'{offset:010d} 00000 n \n'.encode()
    output += f'trailer\n<< /Size {len(offsets)} /Root 1 0 R >>\nstartxref\n{start}\n%%EOF\n'.encode()
    return bytes(output)

def stream(contents, extra=b''):
    return b'<< /Length ' + str(len(contents)).encode() + b' ' + extra + b' >>\nstream\n' + contents + b'\nendstream'

native = 'Plano PR-008 estructura metálica. Biblioteca de prueba y conservación documental.'
scanned = 'DOCUMENTO ESCANEADO. Archivo de planos y memoria técnica para consulta local.'
def text_pdf(text):
    commands = ('BT /F1 18 Tf 48 730 Td (' + text[:43] + ') Tj 0 -32 Td (' + text[43:] + ') Tj ET').encode('cp1252')
    return pdf([
        b'<< /Type /Catalog /Pages 2 0 R >>',
        b'<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
        b'<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
        b'<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>',
        stream(commands),
    ])
(ROOT / 'native.pdf').write_bytes(text_pdf(native))
with tempfile.TemporaryDirectory() as directory:
    source = Path(directory) / 'source.pdf'
    source.write_bytes(text_pdf(scanned))
    image = subprocess.check_output(['pdftoppm', '-r', '150', '-singlefile', str(source)])
    header = re.match(rb'P6\s+(\d+)\s+(\d+)\s+255\s', image)
    assert header
    width, height = header.group(1), header.group(2)
    pixels = image[header.end():]
    assert len(pixels) == int(width) * int(height) * 3
    image_object = stream(zlib.compress(pixels), b'/Type /XObject /Subtype /Image /Width ' + width + b' /Height ' + height + b' /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode')
    objects = [
        b'<< /Type /Catalog /Pages 2 0 R >>',
        b'<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
        b'<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /XObject << /Im1 4 0 R >> >> /Contents 5 0 R >>',
        image_object,
        stream(b'q 612 0 0 792 0 0 cm /Im1 Do Q'),
    ]
    (ROOT / 'scanned.pdf').write_bytes(pdf(objects))
print('Synthetic fixtures:', ', '.join(path.name for path in ROOT.glob('*.pdf')))
